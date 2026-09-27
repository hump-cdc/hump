// Package livestreamer decorates the operations collector with the live feed
// the web UI's SSE stream carries. Deliveries accumulate into a window that
// closes on a timer, so a pipeline running at thousands of rows a second still
// reaches the browser as about one event per second.
package livestreamer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	port "hump/internal/port/live_stream"
	operationscollector "hump/internal/port/operations_collector"
	"hump/pkg/app/shutdown"
)

const (
	// defaultInterval is how long deliveries accumulate before the window closes.
	defaultInterval = 500 * time.Millisecond
	// subscriberBuffer is how many windows a subscriber may fall behind by
	// before it starts losing them. At the default interval it is at least four
	// seconds, which brackets huma's five-second SSE write timeout.
	subscriberBuffer = 8
)

// Scraper refreshes the dashboard metrics out of schedule, closing the returned
// channel once they are. It is the part of the metrics collector this package
// needs, and is nil when telemetry is off.
type Scraper interface {
	Scrape() <-chan struct{}
}

// Streamer wraps an operations collector with the live feed. Every method of
// the collector delegates to the wrapped one; Record additionally adds the
// delivery to the open window when a client is watching the feed.
type Streamer struct {
	inner    operationscollector.Collector
	scraper  Scraper // scraper is nil when telemetry is disabled.
	interval time.Duration

	watched atomic.Bool // watched mirrors len(subs) > 0, so Record costs one load when nobody is watching.

	mu         sync.Mutex
	subs       map[uint64]*subscriber
	nextID     uint64
	pending    []port.Operation
	hint       int // hint sizes the next window from the last one, so a steady stream allocates once per window.
	timer      *time.Timer
	generation uint64 // generation invalidates a timer that fired before the window it was armed for was dropped.
	closed     bool
	done       chan struct{} // done is closed with the streamer, releasing a flush waiting on a scrape.
}

// subscriber is one connected client's channel and its drop bookkeeping.
type subscriber struct {
	ch    chan port.Batch
	drops int // drops counts the windows this subscriber was too slow to take.
}

// Option configures a Streamer.
type Option func(*Streamer)

// WithInterval sets how long deliveries accumulate before the window closes. A
// non-positive d leaves the default.
func WithInterval(d time.Duration) Option {
	return func(s *Streamer) {
		if d > 0 {
			s.interval = d
		}
	}
}

// WithScraper sets the metrics collector a closing window refreshes. A nil one
// leaves the feed unscraped.
func WithScraper(scraper Scraper) Option {
	return func(s *Streamer) {
		if scraper != nil {
			s.scraper = scraper
		}
	}
}

// New returns a streamer wrapping inner and registers its shutdown hook.
func New(inner operationscollector.Collector, opts ...Option) *Streamer {
	s := &Streamer{
		inner:    inner,
		interval: defaultInterval,
		subs:     make(map[uint64]*subscriber),
		done:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	shutdown.OnShutdown(shutdown.StageWorkers, "live_streamer", s.onShutdown)

	return s
}

// Connect registers a client on the wrapped collector.
func (s *Streamer) Connect() { s.inner.Connect() }

// Disconnect unregisters a client on the wrapped collector.
func (s *Streamer) Disconnect() { s.inner.Disconnect() }

// Connected reports whether the wrapped collector is recording.
func (s *Streamer) Connected() bool { return s.inner.Connected() }

// Dump returns a window of the wrapped collector's log.
func (s *Streamer) Dump(start, end int) operationscollector.Page {
	return s.inner.Dump(start, end)
}

// Search returns the window of the wrapped collector's log matching q.
func (s *Streamer) Search(q operationscollector.Query) operationscollector.Page {
	return s.inner.Search(q)
}

// Record stores rec in the wrapped collector and, while a client is watching
// the live feed, adds it to the open window. The five strings are read before
// the collector takes rec over, so nothing here depends on what its buffer swap
// leaves behind.
func (s *Streamer) Record(rec *operationscollector.Record) {
	if !s.watched.Load() {
		s.inner.Record(rec)

		return
	}

	op := port.Operation{
		Schema: string(rec.Table.Schema),
		Table:  string(rec.Table.Name),
		Sink:   rec.SinkType,
		Host:   rec.Host,
		Target: rec.Target,
	}

	s.inner.Record(rec)
	s.append(op)
}

// Subscribe registers a client and returns the channel its windows arrive on.
func (s *Streamer) Subscribe() (uint64, <-chan port.Batch) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		ch := make(chan port.Batch)
		close(ch)

		return 0, ch
	}

	id := s.nextID
	s.nextID++

	sub := &subscriber{ch: make(chan port.Batch, subscriberBuffer)}
	s.subs[id] = sub
	s.watched.Store(true)

	return id, sub.ch
}

// Unsubscribe removes the subscriber with id and closes its channel. Losing the
// last one drops the open window too, so nothing accumulates for a feed nobody
// reads. It is safe to call more than once.
func (s *Streamer) Unsubscribe(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subs[id]
	if !ok {
		return
	}

	delete(s.subs, id)
	close(sub.ch)

	if sub.drops > 0 {
		logger.Warn("Subscriber left after losing windows", "subscriber", id, "dropped", sub.drops)
	}

	if len(s.subs) == 0 {
		s.watched.Store(false)
		s.discard()
	}
}

// append adds op to the open window, arming the flush timer when the window was
// empty. It does nothing once the last subscriber has left, which watched only
// reports approximately.
func (s *Streamer) append(op port.Operation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || len(s.subs) == 0 {
		return
	}

	if s.pending == nil {
		s.pending = make([]port.Operation, 0, s.hint)
	}
	s.pending = append(s.pending, op)

	if s.timer == nil {
		s.arm()
	}
}

// arm starts the timer that closes the open window. s.mu must be held.
func (s *Streamer) arm() {
	s.generation++
	generation := s.generation
	s.timer = time.AfterFunc(s.interval, func() { s.flush(generation) })
}

// flush closes the window, waits for the dashboard metrics to be scraped, and
// hands the batch to every subscriber, skipping any whose buffer is full.
// generation identifies the window the timer was armed for: one that fired
// just before its window was dropped must leave the next window alone.
//
// The batch goes out only after the scrape, so a client refetching the metrics
// on it reads counts that include it. The wait happens off s.mu, keeping it off
// the delivery path; s.timer stays set meanwhile, so no other window opens and
// deliveries that arrive in the meantime go out as the next batch, in order.
// The cost is latency: an event can trail its first delivery by the interval
// plus the collector's debounce.
//
// The fan-out runs under the lock on purpose. The sends never block, and
// Unsubscribe closes a channel under the same lock, so fanning out from a
// snapshot taken outside it would send on a closed channel.
func (s *Streamer) flush(generation uint64) {
	s.mu.Lock()

	if generation != s.generation {
		s.mu.Unlock()
		return
	}

	batch := port.Batch{Operations: s.pending}
	s.hint = len(s.pending)
	s.pending = nil

	if len(batch.Operations) == 0 {
		s.timer = nil
		s.mu.Unlock()
		return
	}

	s.mu.Unlock()

	if s.scraper != nil {
		select {
		case <-s.scraper.Scrape():
		case <-s.done:
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// The window was dropped while waiting, by the last subscriber leaving or
	// by close; the batch goes with it.
	if generation != s.generation {
		return
	}

	s.timer = nil

	for id, sub := range s.subs {
		select {
		case sub.ch <- batch:
		default:
			sub.drops++
			if sub.drops == 1 {
				logger.Warn("Dropping windows for a slow subscriber", "subscriber", id, "buffer", subscriberBuffer)
			}
		}
	}

	if len(s.pending) > 0 {
		s.arm()
	}
}

// discard drops the open window and disarms its timer, bumping the generation
// so a timer that already fired leaves the next window alone. s.mu must be held.
func (s *Streamer) discard() {
	s.generation++

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	s.pending = nil
}

// onShutdown implements the shutdown hook registered by New.
func (s *Streamer) onShutdown(context.Context) error {
	s.close()

	return nil
}

// close stops the feed for good: every subscriber's channel is closed so its
// SSE handler returns, and the open window is dropped. A later Record still
// reaches the wrapped collector. It is idempotent.
func (s *Streamer) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true
	s.watched.Store(false)
	close(s.done)

	s.discard()

	for id, sub := range s.subs {
		delete(s.subs, id)
		close(sub.ch)
	}
}

// Compile-time proof the decorator is substitutable for what it wraps.
var (
	_ operationscollector.Collector = (*Streamer)(nil)
	_ port.Streamer                 = (*Streamer)(nil)
)

// Package streamer fans a single stream of operation events out to many SSE
// subscribers. Events arrive on one unbuffered channel and are delivered to each
// subscriber's buffered channel; when a subscriber's buffer is full the event is
// dropped for that subscriber, so a slow client never blocks the producer.
package streamer

import (
	"context"
	"sync"

	"hump/internal/app/shutdown"
	"hump/internal/types/stream"
)

// subscriberBuffer is the per-subscriber channel capacity. Events are dropped
// for a subscriber whose buffer is full.
const subscriberBuffer = 64

// FanOutStreamer implements stream.Publisher by fanning events out to every
// active subscriber. Each event is an stream.Event or stream.RetryEvent.
type FanOutStreamer struct {
	in chan stream.Message

	mu     sync.RWMutex
	out    map[uint64]chan stream.Message
	nextID uint64
	closed bool
}

// New creates a FanOutStreamer, starts its fan-out goroutine, and registers a
// shutdown hook that tears it down when the application stops.
func New() *FanOutStreamer {
	s := &FanOutStreamer{
		in:  make(chan stream.Message),
		out: make(map[uint64]chan stream.Message),
	}
	go s.run()

	shutdown.OnShutdown("streamer", func(_ context.Context) error {
		s.close()
		return nil
	})

	return s
}

// run delivers each incoming event to every subscriber, skipping any whose
// buffer is full. It returns once the in channel is closed.
func (s *FanOutStreamer) run() {
	for event := range s.in {
		s.mu.RLock()
		for id, ch := range s.out {
			select {
			case ch <- event:
			default:
				// Slow subscriber: its buffer is full, so drop this event for it.
				logger.Warn("dropped event for slow subscriber",
					"subscriber", id, "buffer", subscriberBuffer)
			}
		}
		s.mu.RUnlock()
	}
}

// Publish hands an event (an stream.Event or stream.RetryEvent) to the fan-out
// goroutine. It blocks only until that goroutine receives, which is fast since
// fan-out itself never blocks. After the streamer is closed the event is
// dropped, so producers that outlive shutdown never send on a closed channel.
func (s *FanOutStreamer) Publish(event stream.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return
	}

	s.in <- event
}

// HasSubscribers reports whether at least one client is currently subscribed.
func (s *FanOutStreamer) HasSubscribers() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.out) > 0
}

// Subscribe registers a new subscriber and returns its id together with the
// channel that receives fanned-out events. Call Unsubscribe with the id when the
// client disconnects. If the streamer is already closed it returns a closed
// channel so the caller exits immediately.
func (s *FanOutStreamer) Subscribe() (uint64, <-chan stream.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		ch := make(chan stream.Message)
		close(ch)
		return 0, ch
	}

	id := s.nextID
	s.nextID++

	ch := make(chan stream.Message, subscriberBuffer)
	s.out[id] = ch

	return id, ch
}

// Unsubscribe removes the subscriber with the given id and closes its channel.
// It is safe to call more than once.
func (s *FanOutStreamer) Unsubscribe(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.out[id]; ok {
		delete(s.out, id)
		close(ch)
	}
}

// close stops the fan-out goroutine and closes every subscriber channel. Closing
// in ends run's range loop, so the goroutine returns on its own; closing each
// subscriber channel makes its SSE handler observe !ok and return. It is
// idempotent.
func (s *FanOutStreamer) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	close(s.in)
	for id, ch := range s.out {
		delete(s.out, id)
		close(ch)
	}
}

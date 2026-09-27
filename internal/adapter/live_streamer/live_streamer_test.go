package livestreamer

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hump/internal/domain"
	port "hump/internal/port/live_stream"
	operationscollector "hump/internal/port/operations_collector"

	"go.uber.org/goleak"
)

func TestStreamerDelegatesToTheWrappedCollector(t *testing.T) {
	inner := &fakeCollector{}
	s := newTestStreamer(t, inner)

	s.Connect()
	s.Disconnect()

	if inner.connects != 1 || inner.disconnects != 1 {
		t.Fatalf("connects, disconnects = %d, %d, want 1, 1", inner.connects, inner.disconnects)
	}

	inner.connected = true
	if !s.Connected() {
		t.Fatal("Connected = false, want the wrapped collector's answer")
	}

	inner.page = operationscollector.Page{Total: 7}
	if got := s.Dump(0, 1).Total; got != 7 {
		t.Fatalf("Dump total = %d, want 7", got)
	}

	if got := s.Search(operationscollector.Query{SinkTypes: []string{"kafka"}}).Total; got != 7 {
		t.Fatalf("Search total = %d, want 7", got)
	}
	if !slices.Equal(inner.query.SinkTypes, []string{"kafka"}) {
		t.Fatalf("Search passed %q, want the query it was given", inner.query.SinkTypes)
	}
}

func TestStreamerSendsOneBatchPerWindow(t *testing.T) {
	inner := &fakeCollector{}
	s := newTestStreamer(t, inner, WithInterval(100*time.Millisecond))

	_, batches := s.Subscribe()
	write(s, "a")
	write(s, "b")
	write(s, "c")

	select {
	case <-batches:
		t.Fatal("a batch arrived before the window closed")
	case <-time.After(20 * time.Millisecond):
	}

	batch := receive(t, batches)
	if got := targets(batch); !equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("batch = %v, want [a b c] in the order they were recorded", got)
	}

	if inner.records.Load() != 3 {
		t.Fatalf("the wrapped collector saw %d records, want 3", inner.records.Load())
	}
}

func TestStreamerOpensANewWindowAfterAFlush(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	_, batches := s.Subscribe()
	write(s, "a")
	first := receive(t, batches)

	write(s, "b")
	second := receive(t, batches)

	if got := targets(second); !equal(got, []string{"b"}) {
		t.Fatalf("second batch = %v, want [b]", got)
	}
	if got := targets(first); !equal(got, []string{"a"}) {
		t.Fatalf("first batch = %v after the second arrived, want [a] — the window was reused", got)
	}
}

func TestStreamerBuffersNothingWithoutSubscribers(t *testing.T) {
	inner := &fakeCollector{}
	s := newTestStreamer(t, inner)

	for range 100 {
		write(s, "a")
	}

	time.Sleep(20 * time.Millisecond)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pending != nil || s.timer != nil {
		t.Fatalf("pending, timer = %v, %v with no subscriber, want nil, nil", s.pending, s.timer)
	}
	if inner.records.Load() != 100 {
		t.Fatalf("the wrapped collector saw %d records, want 100", inner.records.Load())
	}
}

func TestStreamerFansOutToEverySubscriber(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	_, first := s.Subscribe()
	_, second := s.Subscribe()
	write(s, "a")

	if got := targets(receive(t, first)); !equal(got, []string{"a"}) {
		t.Fatalf("first subscriber got %v, want [a]", got)
	}
	if got := targets(receive(t, second)); !equal(got, []string{"a"}) {
		t.Fatalf("second subscriber got %v, want [a]", got)
	}
}

func TestStreamerDropsWindowsForASlowSubscriber(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	slow, _ := s.Subscribe()
	_, healthy := s.Subscribe()

	for range subscriberBuffer + 2 {
		write(s, "a")
		receive(t, healthy)
	}

	s.mu.Lock()
	drops := s.subs[slow].drops
	s.mu.Unlock()

	if drops == 0 {
		t.Fatal("the slow subscriber lost no window, want at least one")
	}
}

func TestStreamerScrapesTheMetricsBeforeAWindowGoesOut(t *testing.T) {
	scraper := &fakeScraper{}
	s := newTestStreamer(t, &fakeCollector{}, WithScraper(scraper))

	_, batches := s.Subscribe()
	write(s, "a")
	receive(t, batches)

	if got := scraper.calls.Load(); got != 1 {
		t.Fatalf("Scrape was called %d times for one window, want 1", got)
	}
}

func TestStreamerLastUnsubscribeDropsTheOpenWindow(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	id, _ := s.Subscribe()
	write(s, "a")
	s.Unsubscribe(id)

	time.Sleep(20 * time.Millisecond)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pending != nil || s.timer != nil {
		t.Fatalf("pending, timer = %v, %v after the last subscriber left, want nil, nil", s.pending, s.timer)
	}
	if s.watched.Load() {
		t.Fatal("watched = true after the last subscriber left")
	}
}

func TestStreamerIgnoresAStaleTimer(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	_, batches := s.Subscribe()
	write(s, "a")

	s.mu.Lock()
	generation := s.generation
	s.mu.Unlock()

	s.flush(generation - 1)

	s.mu.Lock()
	armed := s.timer != nil
	s.mu.Unlock()

	if !armed {
		t.Fatal("a stale flush disarmed the open window's timer")
	}

	if got := targets(receive(t, batches)); !equal(got, []string{"a"}) {
		t.Fatalf("batch = %v, want [a] — the stale flush published the open window", got)
	}
}

func TestStreamerUnsubscribeClosesTheChannel(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	id, batches := s.Subscribe()
	s.Unsubscribe(id)
	s.Unsubscribe(id)

	if _, ok := <-batches; ok {
		t.Fatal("the channel of an unsubscribed client is still open")
	}
}

func TestStreamerCloseClosesEverySubscriber(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	_, first := s.Subscribe()
	_, second := s.Subscribe()

	s.close()
	s.close()

	if _, ok := <-first; ok {
		t.Fatal("the first subscriber's channel survived close")
	}
	if _, ok := <-second; ok {
		t.Fatal("the second subscriber's channel survived close")
	}
}

func TestStreamerSubscribeAfterCloseAnswersAClosedChannel(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})
	s.close()

	_, batches := s.Subscribe()
	if _, ok := <-batches; ok {
		t.Fatal("a subscriber that arrived after close got an open channel")
	}
}

func TestStreamerRecordAfterCloseStillReachesTheCollector(t *testing.T) {
	inner := &fakeCollector{}
	s := newTestStreamer(t, inner)

	s.Subscribe()
	s.close()
	write(s, "a")

	if inner.records.Load() != 1 {
		t.Fatalf("the wrapped collector saw %d records after close, want 1", inner.records.Load())
	}
}

func TestStreamerRecordDoesNotRetainTheRecordsBuffers(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	_, batches := s.Subscribe()

	rec := newTestRecord("a")
	s.Record(&rec)

	rec.Table.Name = "mutated"
	rec.Target = "mutated"
	rec.Payload = append(rec.Payload[:0], "mutated"...)

	batch := receive(t, batches)
	if got := batch.Operations[0].Table; got != "a" {
		t.Fatalf("the published table is %q, want a — the window aliased the record", got)
	}
	if got := batch.Operations[0].Target; got != "a" {
		t.Fatalf("the published target is %q, want a — the window aliased the record", got)
	}
}

func TestStreamerRecordDoesNotAllocateWithoutSubscribers(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})
	rec := newTestRecord("a")

	if got := testing.AllocsPerRun(1000, func() { s.Record(&rec) }); got != 0 {
		t.Fatalf("Record allocated %v times per call with no subscriber, want 0", got)
	}
}

func TestStreamerKeepsTheDefaultIntervalForANonPositiveOne(t *testing.T) {
	s := New(&fakeCollector{}, WithInterval(0))
	t.Cleanup(s.close)

	if s.interval != defaultInterval {
		t.Fatalf("interval = %v after WithInterval(0), want the default %v", s.interval, defaultInterval)
	}

	s.interval = defaultInterval
	if s = New(&fakeCollector{}, WithInterval(-time.Second)); s.interval != defaultInterval {
		t.Fatalf("interval = %v after a negative WithInterval, want the default %v", s.interval, defaultInterval)
	}
	t.Cleanup(s.close)
}

func TestStreamerSurvivesConcurrentRecordsAndSubscribers(t *testing.T) {
	s := newTestStreamer(t, &fakeCollector{})

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 500 {
				write(s, "a")
			}
		}()
	}

	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				id, ch := s.Subscribe()
				go func() {
					for range ch {
					}
				}()
				s.Unsubscribe(id)
			}
		}()
	}

	wg.Wait()
}

func TestStreamerHasNoBackgroundGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t)

	s := newTestStreamer(t, &fakeCollector{})

	_, batches := s.Subscribe()
	write(s, "a")
	receive(t, batches)
	s.close()
}

// fakeCollector is the wrapped collector, counting what the decorator delegates
// to it and answering Dump and Search with one canned page.
type fakeCollector struct {
	connects    int
	disconnects int
	records     atomic.Int64 // records is atomic: the concurrency test writes it from every recorder.
	connected   bool
	page        operationscollector.Page
	query       operationscollector.Query
}

func (c *fakeCollector) Connect()                           { c.connects++ }
func (c *fakeCollector) Disconnect()                        { c.disconnects++ }
func (c *fakeCollector) Connected() bool                    { return c.connected }
func (c *fakeCollector) Record(*operationscollector.Record) { c.records.Add(1) }

func (c *fakeCollector) Dump(int, int) operationscollector.Page { return c.page }

func (c *fakeCollector) Search(q operationscollector.Query) operationscollector.Page {
	c.query = q

	return c.page
}

// fakeScraper counts the out-of-schedule scrapes a closing window asks for.
type fakeScraper struct {
	calls atomic.Int64
}

func (s *fakeScraper) Scrape() { s.calls.Add(1) }

// newTestStreamer returns a streamer with a short window that is closed when the
// test ends, bypassing the global shutdown hook New registers.
func newTestStreamer(t *testing.T, inner operationscollector.Collector, opts ...Option) *Streamer {
	t.Helper()

	s := New(inner, append([]Option{WithInterval(time.Millisecond)}, opts...)...)
	t.Cleanup(s.close)

	return s
}

// newTestRecord returns a record of one delivery identified by target.
func newTestRecord(target string) operationscollector.Record {
	return operationscollector.Record{
		Table:     domain.TableIdentifier{Schema: "public", Name: domain.TableName(target)},
		Operation: domain.OperationTypeInsert,
		SinkType:  "kafka",
		Host:      "localhost:9092",
		Target:    target,
		Status:    domain.DeliveryStatusSuccess,
		Payload:   []byte(target),
	}
}

// write records one delivery identified by target.
func write(s *Streamer, target string) {
	rec := newTestRecord(target)
	s.Record(&rec)
}

// receive takes the next batch, failing t if none arrives within a second.
func receive(t *testing.T, batches <-chan port.Batch) port.Batch {
	t.Helper()

	select {
	case batch, ok := <-batches:
		if !ok {
			t.Fatal("the subscriber's channel was closed before a batch arrived")
		}

		return batch
	case <-time.After(time.Second):
		t.Fatal("no batch arrived within a second")
	}

	return port.Batch{}
}

// targets lists the targets of a batch, in the order it carries them.
func targets(batch port.Batch) []string {
	out := make([]string, 0, len(batch.Operations))
	for _, op := range batch.Operations {
		out = append(out, op.Target)
	}

	return out
}

// equal reports whether two target lists match.
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

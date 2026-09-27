package operationscollector

import (
	"context"
	"testing"
	"time"

	"hump/internal/domain"
	port "hump/internal/port/operations_collector"

	"go.uber.org/goleak"
)

func TestCollectorIdleUntilConnect(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))

	if c.Connected() {
		t.Fatal("Connected() = true, want false before Connect")
	}

	rec := newTestRecord("a")
	c.Record(&rec)

	if c.ring != nil {
		t.Fatalf("ring = %v, want nil while idle", c.ring)
	}

	if page := c.Dump(0, 10); len(page.Records) != 0 || page.Total != 0 {
		t.Fatalf("Dump() = %+v, want an empty page", page)
	}
}

func TestCollectorAllocatesRingOnConnect(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))
	c.Connect()

	if !c.Connected() || c.ring == nil {
		t.Fatalf("Connected() = %v, ring = %v, want true and a ring", c.Connected(), c.ring)
	}

	if got := len(c.ring.slots); got != 4 {
		t.Fatalf("len(slots) = %d, want 4", got)
	}
}

func TestCollectorDropsRingAfterIdleTimeout(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4), WithIdleTimeout(time.Millisecond))
	c.Connect()
	write(c, "a")
	c.Disconnect()

	waitFor(t, func() bool { return !c.Connected() })

	c.mu.Lock()
	if c.ring != nil {
		c.mu.Unlock()
		t.Fatal("ring is still allocated after the idle timeout")
	}
	c.mu.Unlock()

	c.Connect()
	write(c, "b")

	if got := c.Dump(0, 1).Records[0].Seq; got <= 1 {
		t.Fatalf("Seq = %d after a reconnect, want it past the 1 handed out before the idle gap", got)
	}
}

func TestCollectorStaysActiveWhileClientsRemain(t *testing.T) {
	c := newTestCollector(t, WithIdleTimeout(time.Millisecond))
	c.Connect()
	c.Connect()
	c.Disconnect()

	time.Sleep(20 * time.Millisecond)

	if !c.Connected() {
		t.Fatal("Connected() = false, want true while a client remains")
	}
}

func TestCollectorReconnectCancelsIdleTimeout(t *testing.T) {
	c := newTestCollector(t, WithIdleTimeout(50*time.Millisecond))
	c.Connect()
	c.Disconnect()
	c.Connect()

	time.Sleep(80 * time.Millisecond)

	if !c.Connected() {
		t.Fatal("Connected() = false, want true after a reconnect")
	}
}

func TestCollectorIgnoresConnectAfterClose(t *testing.T) {
	c := newTestCollector(t)
	c.close()
	c.Connect()

	if c.Connected() {
		t.Fatal("Connected() = true, want false after close")
	}
}

func TestCollectorShutdownHookStopsRecording(t *testing.T) {
	c := newTestCollector(t)
	c.Connect()
	write(c, "a")

	if err := c.onShutdown(context.Background()); err != nil {
		t.Fatalf("onShutdown() = %v, want nil", err)
	}

	if c.Connected() {
		t.Fatal("Connected() = true, want false after the shutdown hook ran")
	}
}

func TestRingOverwritesOldest(t *testing.T) {
	c := newTestCollector(t, WithCapacity(3))
	c.Connect()

	for _, target := range []string{"a", "b", "c", "d", "e"} {
		write(c, target)
	}

	page := c.Dump(0, 10)

	if got := targets(page); !equal(got, []string{"e", "d", "c"}) {
		t.Fatalf("targets = %v, want [e d c]", got)
	}

	if page.Total != 3 || page.Written != 5 {
		t.Fatalf("Total, Written = %d, %d, want 3, 5", page.Total, page.Written)
	}

	for i := 1; i < len(page.Records); i++ {
		if page.Records[i-1].Seq <= page.Records[i].Seq {
			t.Fatalf("Seq %d is not newer than %d", page.Records[i-1].Seq, page.Records[i].Seq)
		}
	}
}

func TestDumpPagesFromNewest(t *testing.T) {
	c := newTestCollector(t, WithCapacity(30))
	c.Connect()

	for _, target := range sequence(30) {
		write(c, target)
	}

	whole := targets(c.Dump(0, 30))

	var paged []string
	for start := 0; start < 30; start += 10 {
		paged = append(paged, targets(c.Dump(start, start+10))...)
	}

	if !equal(whole, paged) {
		t.Fatalf("paged = %v, want %v", paged, whole)
	}
}

func TestDumpClampsOffsets(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))
	c.Connect()
	write(c, "a")
	write(c, "b")

	page := c.Dump(-5, 100)
	if page.Start != 0 || page.End != 2 || len(page.Records) != 2 {
		t.Fatalf("Dump(-5, 100) = %+v, want the two records clamped to [0, 2)", page)
	}

	if page := c.Dump(5, 10); len(page.Records) != 0 || page.Total != 2 {
		t.Fatalf("Dump(5, 10) = %+v, want an empty page with Total 2", page)
	}
}

func TestSearchFiltersThePage(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))
	c.Connect()
	write(c, "a")
	write(c, "b")

	page := c.Search(port.Query{Text: "b", End: 10})

	if got := targets(page); !equal(got, []string{"b"}) {
		t.Fatalf("targets = %v, want [b]", got)
	}

	if page.Total != 1 || page.Written != 2 {
		t.Fatalf("Total, Written = %d, %d, want 1 match out of 2 records written", page.Total, page.Written)
	}
}

func TestSearchAnswersAnEmptyPageWhileIdle(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))

	if page := c.Search(port.Query{End: 10}); len(page.Records) != 0 || page.Total != 0 {
		t.Fatalf("Search() = %+v, want an empty page", page)
	}
}

func TestRecordSwapsBuffers(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))
	c.Connect()

	rec := newTestRecord("a")
	rec.Payload = append(rec.Payload, "first"...)
	rec.Attempts = append(rec.Attempts, port.Attempt{Status: domain.DeliveryStatusSuccess})
	rec.Metadata["k"] = "v"
	c.Record(&rec)

	if len(rec.Payload) != 0 || len(rec.Attempts) != 0 {
		t.Fatalf("caller kept %d payload bytes and %d attempts, want the slot's empty buffers", len(rec.Payload), len(rec.Attempts))
	}

	rec.Payload = append(rec.Payload, "second"...)
	rec.Attempts = append(rec.Attempts, port.Attempt{Status: domain.DeliveryStatusFailure})
	clear(rec.Metadata)

	stored := c.Dump(0, 1).Records[0]
	if string(stored.Payload) != "first" || len(stored.Attempts) != 1 || stored.Metadata["k"] != "v" {
		t.Fatalf("stored record = %q, %v, %v, want the first write untouched", stored.Payload, stored.Attempts, stored.Metadata)
	}
}

func TestDumpReturnsCopies(t *testing.T) {
	c := newTestCollector(t, WithCapacity(4))
	c.Connect()
	write(c, "a")

	page := c.Dump(0, 1)
	page.Records[0].Payload[0] = 'X'
	page.Records[0].Metadata["k"] = "mutated"

	again := c.Dump(0, 1).Records[0]
	if again.Payload[0] == 'X' || again.Metadata["k"] == "mutated" {
		t.Fatal("Dump handed out the ring's own buffers")
	}
}

func TestRecordTruncatesOversizedPayload(t *testing.T) {
	c := newTestCollector(t, WithCapacity(1), WithPayloadLimit(8))
	c.Connect()

	// Prime the slot so it has a buffer to hand back; on an untouched slot the
	// assertion below would pass on a nil one.
	write(c, "abc")

	rec := newTestRecord("a")
	rec.Payload = append(rec.Payload, "0123456789abcdef"...)
	c.Record(&rec)

	stored := c.Dump(0, 1).Records[0]
	if string(stored.Payload) != "01234567" || stored.PayloadSize != 16 {
		t.Fatalf("stored = %q with size %d, want %q with size 16", stored.Payload, stored.PayloadSize, "01234567")
	}

	if cap(rec.Payload) == 0 || cap(rec.Payload) > 8 {
		t.Fatalf("caller got a %d byte buffer back, want the slot's previous one, within the limit", cap(rec.Payload))
	}
}

func TestCollectorHasNoBackgroundGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t)

	c := newTestCollector(t, WithIdleTimeout(time.Millisecond))
	c.Connect()
	write(c, "a")
	c.Disconnect()

	waitFor(t, func() bool { return !c.Connected() })
}

// newTestCollector returns a collector that is closed when the test ends,
// bypassing the global shutdown hook New registers.
func newTestCollector(t *testing.T, opts ...Option) *Collector {
	t.Helper()

	c := New(opts...)
	t.Cleanup(c.close)

	return c
}

// newTestRecord returns a record identified by target, with the buffers a
// caller would hand in.
func newTestRecord(target string) port.Record {
	return port.Record{
		StartedAt: time.Now(),
		Table:     domain.TableIdentifier{Schema: "public", Name: "users"},
		Operation: domain.OperationTypeInsert,
		SinkType:  "kafka",
		Target:    target,
		Status:    domain.DeliveryStatusSuccess,
		Payload:   []byte{},
		Metadata:  map[string]string{},
	}
}

// write records one delivery identified by target.
func write(c *Collector, target string) {
	rec := newTestRecord(target)
	rec.Payload = append(rec.Payload, target...)
	rec.Metadata["k"] = "v"
	c.Record(&rec)
}

// targets lists the targets of a page, in the order the page reports them.
func targets(page port.Page) []string {
	out := make([]string, 0, len(page.Records))
	for _, rec := range page.Records {
		out = append(out, rec.Target)
	}

	return out
}

// sequence returns n distinct targets, "0" through "n-1".
func sequence(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, string(rune('a'+i)))
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

// waitFor polls cond at 1 ms for up to a second.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition was not met within a second")
}

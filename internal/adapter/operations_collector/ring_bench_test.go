package operationscollector

import (
	"runtime"
	"strconv"
	"testing"

	port "hump/internal/port/operations_collector"
)

func TestRingWrite1MRecordsStaysBounded(t *testing.T) {
	const (
		records = 1_000_000
		// heapLimit and allocLimit are generous: the point is that neither
		// grows with the number of writes, not the exact steady-state figure.
		heapLimit  = 4 << 20
		allocLimit = records / 1000
	)

	r := newRing(1000, defaultPayloadLimit, 0)
	rec := ringRecord("users", 256)

	// The first lap allocates the buffers every later write reuses, so it is
	// primed before the measurement rather than counted in it.
	for range len(r.slots) {
		refillRingRecord(&rec)
		r.write(&rec)
	}

	before := heapStats()

	for range records - len(r.slots) {
		refillRingRecord(&rec)
		r.write(&rec)
	}

	after := heapStats()

	if r.written != records || r.seq != records {
		t.Fatalf("written, seq = %d, %d, want %d each", r.written, r.seq, records)
	}

	if got := r.total(); got != len(r.slots) {
		t.Fatalf("total() = %d, want %d, the capacity", got, len(r.slots))
	}

	if got := len(r.page(0, records).Records); got != len(r.slots) {
		t.Fatalf("page holds %d records, want the %d newest", got, len(r.slots))
	}

	if grown := int64(after.HeapAlloc) - int64(before.HeapAlloc); grown > heapLimit {
		t.Fatalf("the heap grew by %d bytes over %d writes, want at most %d", grown, records, heapLimit)
	}

	if allocs := after.Mallocs - before.Mallocs; allocs > allocLimit {
		t.Fatalf("%d writes allocated %d times, want at most %d", records, allocs, allocLimit)
	}
}

func BenchmarkRingWrite(b *testing.B) {
	r := newRing(1000, defaultPayloadLimit, 0)
	rec := ringRecord("users", 256)

	// The ring hands back the buffers of the slot it overwrites, so the first
	// lap still allocates: prime every slot before measuring.
	for range len(r.slots) {
		refillRingRecord(&rec)
		r.write(&rec)
	}

	b.ReportAllocs()

	for b.Loop() {
		refillRingRecord(&rec)
		r.write(&rec)
	}
}

func BenchmarkRingWriteOversized(b *testing.B) {
	r := newRing(1000, 1024, 0)
	rec := ringRecord("users", 256)
	// A payload the caller owns for good: refilling the buffer the ring hands
	// back would add an allocation the truncation is not responsible for.
	payload := make([]byte, 4096)

	for range len(r.slots) {
		rec.Payload = payload
		r.write(&rec)
	}

	b.ReportAllocs()

	for b.Loop() {
		rec.Payload = payload
		r.write(&rec)
	}
}

func BenchmarkRingPage(b *testing.B) {
	r := newRing(1000, defaultPayloadLimit, 0)
	rec := ringRecord("users", 256)

	for range len(r.slots) {
		refillRingRecord(&rec)
		r.write(&rec)
	}

	for _, size := range []int{1, 50, 1000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				r.page(0, size)
			}
		})
	}
}

func BenchmarkClone(b *testing.B) {
	rec := ringRecord("users", 256)
	refillRingRecord(&rec)

	b.ReportAllocs()

	for b.Loop() {
		cloneSink = clone(&rec)
	}
}

// heapStats reports the heap after a collection, so a measurement is not the
// garbage of the one before it.
func heapStats() runtime.MemStats {
	runtime.GC()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return stats
}

// cloneSink keeps the clones alive, so the metadata map escapes as it does on
// the page path and the benchmark counts the allocation production pays.
var cloneSink port.Record

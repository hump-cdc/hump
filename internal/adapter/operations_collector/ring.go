package operationscollector

import (
	port "hump/internal/port/operations_collector"
	"maps"
)

// ring is a fixed-capacity buffer of records that overwrites the oldest one.
// Its slots own the Payload, Metadata and Attempts buffers of the records they
// hold, and hand them back to the writer on the next write, so a steady stream
// of deliveries allocates nothing.
type ring struct {
	slots   []port.Record
	next    int    // next is the slot the following write lands in.
	written uint64 // written counts every write since the ring was created, including overwritten ones.
	seq     uint64 // seq numbers records; it carries over from the previous activation so a reader holding one as a cursor never sees it go backwards.
	limit   int    // limit caps the payload bytes one slot may retain.
}

// newRing allocates a ring of capacity slots, each retaining at most limit
// payload bytes, numbering its records from seq.
func newRing(capacity, limit int, seq uint64) *ring {
	return &ring{slots: make([]port.Record, capacity), limit: limit, seq: seq}
}

// write stores rec in the oldest slot and hands the slot's previous buffers
// back to rec, so a steady stream of deliveries allocates nothing. rec keeps
// its scalars but must not be read through its old buffers after the call.
func (r *ring) write(rec *port.Record) {
	slot := &r.slots[r.next]

	r.next++
	// A comparison rather than a remainder: the capacity is configurable and
	// need not be a power of two.
	if r.next == len(r.slots) {
		r.next = 0
	}

	r.written++
	r.seq++

	sparePayload, spareAttempts, spareMetadata := slot.Payload, slot.Attempts, slot.Metadata

	*slot = *rec
	slot.Seq = r.seq
	slot.PayloadSize = len(rec.Payload)

	// An oversized payload is copied down to the limit and its buffer dropped,
	// so one huge row cannot pin memory for the life of the ring. That copy is
	// the only allocation on this path.
	if len(rec.Payload) > r.limit {
		slot.Payload = append(make([]byte, 0, r.limit), rec.Payload[:r.limit]...)
	}

	rec.Payload = sparePayload[:0]
	rec.Attempts = spareAttempts[:0]
	rec.Metadata = spareMetadata
}

// total is how many records the ring currently holds.
func (r *ring) total() int {
	if r.written < uint64(len(r.slots)) {
		return int(r.written)
	}

	return len(r.slots)
}

// page copies the records in [start, end) counted back from the newest. Both
// offsets are clamped to total, so an out-of-range window is empty rather than
// an error. Every buffer is copied, since write reuses the slots afterwards.
func (r *ring) page(start, end int) port.Page {
	total := r.total()
	start = clamp(start, total)
	end = clamp(end, total)

	p := port.Page{Start: start, End: end, Total: total, Written: r.written}
	if start >= end {
		return p
	}

	p.Records = make([]port.Record, 0, end-start)
	for offset := start; offset < end; offset++ {
		p.Records = append(p.Records, clone(&r.slots[r.index(offset)]))
	}

	return p
}

// index maps an offset counted back from the newest record to its slot.
func (r *ring) index(offset int) int {
	i := r.next - 1 - offset
	if i < 0 {
		i += len(r.slots)
	}

	return i
}

// clone deep-copies the buffers of rec so a reader keeps nothing the ring owns.
func clone(rec *port.Record) port.Record {
	out := *rec
	out.Payload = append([]byte(nil), rec.Payload...)
	out.Attempts = append([]port.Attempt(nil), rec.Attempts...)

	if rec.Metadata != nil {
		out.Metadata = make(map[string]string, len(rec.Metadata))
		maps.Copy(out.Metadata, rec.Metadata)
	}

	return out
}

// clamp bounds v to [0, max].
func clamp(v, max int) int {
	if v < 0 {
		return 0
	}

	if v > max {
		return max
	}

	return v
}

package operationscollector

import (
	"maps"
	"slices"
	"strings"

	port "hump/internal/port/operations_collector"
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
	return len(r.slots[:min(r.written, uint64(len(r.slots)))])
}

// page copies the records in [start, end) counted back from the newest.
func (r *ring) page(start, end int) port.Page {
	return r.search(port.Query{Start: start, End: end})
}

// search copies the records matching q in [q.Start, q.End) counted back from
// the newest match. Both offsets are clamped to the number of matches, so an
// out-of-range window is empty rather than an error. Every buffer is copied,
// since write reuses the slots afterwards.
func (r *ring) search(q port.Query) port.Page {
	total := r.total()
	m := newMatcher(q)

	// The whole ring is walked even once the window is filled, because Total
	// has to count every match.
	p := port.Page{Written: r.written}
	for offset := range total {
		rec := &r.slots[r.index(offset)]
		if !m.matches(rec) {
			continue
		}

		if p.Total >= q.Start && p.Total < q.End {
			p.Records = append(p.Records, clone(rec))
		}

		p.Total++
	}

	p.Start = clamp(q.Start, p.Total)
	p.End = clamp(q.End, p.Total)

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

// matcher is a Query prepared for the scan: the parts that would otherwise be
// lowercased once per record are done once here instead, so matching a record
// allocates nothing.
type matcher struct {
	query port.Query
	text  string // text is query.Text lowercased, since containsFold expects a lowercased needle.
}

// newMatcher prepares q for the scan.
func newMatcher(q port.Query) matcher {
	return matcher{query: q, text: strings.ToLower(q.Text)}
}

// matches reports whether rec satisfies every filter of the query.
func (m *matcher) matches(rec *port.Record) bool {
	q := &m.query

	if len(q.SinkTypes) > 0 && !slices.Contains(q.SinkTypes, rec.SinkType) {
		return false
	}

	if len(q.Hosts) > 0 && !slices.Contains(q.Hosts, rec.Host) {
		return false
	}

	if len(q.Targets) > 0 && !slices.Contains(q.Targets, rec.TargetKey) {
		return false
	}

	if len(q.Tables) > 0 && !m.matchesTable(rec) {
		return false
	}

	if len(q.Operations) > 0 && !slices.Contains(q.Operations, rec.Operation) {
		return false
	}

	if len(q.Statuses) > 0 && !slices.Contains(q.Statuses, rec.Status) {
		return false
	}

	return m.text == "" || m.matchesText(rec)
}

// matchesTable reports whether rec's table is one of the query's. Each name is
// compared in place against schema, dot and name, since building
// "schema.name" per record would allocate.
func (m *matcher) matchesTable(rec *port.Record) bool {
	schema, name := string(rec.Table.Schema), string(rec.Table.Name)

	for _, table := range m.query.Tables {
		if len(table) == len(schema)+1+len(name) &&
			table[:len(schema)] == schema && table[len(schema)] == '.' && table[len(schema)+1:] == name {
			return true
		}
	}

	return false
}

// matchesText reports whether the prepared needle occurs anywhere a reader
// would look for it: the payload as far as it was kept, the rendered target,
// the errors, and both halves of every metadata entry.
func (m *matcher) matchesText(rec *port.Record) bool {
	if containsFold(rec.Payload, m.text) || containsFold(rec.Target, m.text) || containsFold(rec.Error, m.text) {
		return true
	}

	for key, value := range rec.Metadata {
		if containsFold(key, m.text) || containsFold(value, m.text) {
			return true
		}
	}

	for i := range rec.Attempts {
		if containsFold(rec.Attempts[i].Error, m.text) {
			return true
		}
	}

	return false
}

// containsFold reports whether hay contains needle, comparing ASCII letters
// case-insensitively. needle must already be lowercased. It is generic over
// string and []byte so a payload needs no conversion, and it allocates
// nothing, which strings.Contains(strings.ToLower(...)) would not manage.
func containsFold[T ~string | ~[]byte](hay T, needle string) bool {
	if len(needle) == 0 {
		return true
	}

	for i := 0; i+len(needle) <= len(hay); i++ {
		if matchesAt(hay, needle, i) {
			return true
		}
	}

	return false
}

// matchesAt reports whether needle occurs in hay at i, folding ASCII case.
func matchesAt[T ~string | ~[]byte](hay T, needle string, i int) bool {
	for j := range len(needle) {
		if lowerASCII(hay[i+j]) != needle[j] {
			return false
		}
	}

	return true
}

// lowerASCII lowercases an ASCII letter and leaves every other byte alone,
// which keeps a UTF-8 continuation byte from being altered.
func lowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}

	return b
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

// clamp bounds v to [0, upper].
func clamp(v, upper int) int {
	if v < 0 {
		return 0
	}

	if v > upper {
		return upper
	}

	return v
}

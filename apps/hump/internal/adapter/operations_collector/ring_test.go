package operationscollector

import (
	"reflect"
	"testing"
	"time"

	"hump/internal/domain"
	port "hump/internal/port/operations_collector"
)

func TestNewRingStartsEmpty(t *testing.T) {
	r := newRing(4, 8, 0)

	if got := len(r.slots); got != 4 {
		t.Fatalf("len(slots) = %d, want 4", got)
	}

	if got := r.total(); got != 0 {
		t.Fatalf("total() = %d, want 0", got)
	}

	page := r.page(0, 10)
	if len(page.Records) != 0 || page.Total != 0 || page.Written != 0 {
		t.Fatalf("page(0, 10) = %+v, want an empty page", page)
	}
}

func TestNewRingResumesNumberingFromSeq(t *testing.T) {
	r := newRing(4, 8, 7)

	rec := ringRecord("a", 8)
	r.write(&rec)

	if got := r.page(0, 1).Records[0].Seq; got != 8 {
		t.Fatalf("Seq = %d, want 8, one past the seq the ring resumed from", got)
	}
}

func TestRingWriteWrapsAndCapsTotal(t *testing.T) {
	r := newRing(3, 8, 0)

	for _, target := range []string{"a", "b", "c", "d", "e"} {
		rec := ringRecord(target, 8)
		r.write(&rec)
	}

	if got := r.total(); got != 3 {
		t.Fatalf("total() = %d, want 3", got)
	}

	if r.written != 5 || r.next != 2 || r.seq != 5 {
		t.Fatalf("written, next, seq = %d, %d, %d, want 5, 2, 5", r.written, r.next, r.seq)
	}

	if got := targets(r.page(0, 3)); !equal(got, []string{"e", "d", "c"}) {
		t.Fatalf("targets = %v, want [e d c]", got)
	}
}

func TestRingWriteWrapsOnEverySlot(t *testing.T) {
	r := newRing(1, 8, 0)

	for _, target := range []string{"a", "b", "c"} {
		rec := ringRecord(target, 8)
		r.write(&rec)
	}

	if r.next != 0 || r.total() != 1 || r.written != 3 {
		t.Fatalf("next, total, written = %d, %d, %d, want 0, 1, 3", r.next, r.total(), r.written)
	}

	if got := targets(r.page(0, 1)); !equal(got, []string{"c"}) {
		t.Fatalf("targets = %v, want [c]", got)
	}
}

func TestRingWriteHandsBackNilBuffersOnTheFirstLap(t *testing.T) {
	r := newRing(2, 8, 0)

	rec := ringRecord("a", 8)
	r.write(&rec)

	if rec.Payload != nil || rec.Attempts != nil || rec.Metadata != nil {
		t.Fatalf("caller got %v, %v, %v back, want the untouched slot's nil buffers", rec.Payload, rec.Attempts, rec.Metadata)
	}
}

func TestRingWriteHandsBackTheOverwrittenSlotsBuffers(t *testing.T) {
	r := newRing(1, 64, 0)

	first := ringRecord("a", 64)
	firstPayload, firstAttempts, firstMetadata := first.Payload, first.Attempts, first.Metadata
	r.write(&first)

	second := ringRecord("b", 64)
	r.write(&second)

	if !samePayload(second.Payload, firstPayload) || !sameAttempts(second.Attempts, firstAttempts) {
		t.Fatal("caller got fresh buffers back, want the ones the overwritten slot held")
	}

	if len(second.Payload) != 0 || len(second.Attempts) != 0 {
		t.Fatalf("buffers came back with %d payload bytes and %d attempts, want them emptied", len(second.Payload), len(second.Attempts))
	}

	if len(second.Metadata) != 1 || second.Metadata["k"] != "a" {
		t.Fatalf("Metadata = %v, want the overwritten slot's map, handed back uncleared", second.Metadata)
	}

	if !sameMetadata(second.Metadata, firstMetadata) {
		t.Fatal("Metadata is not the map the overwritten slot held")
	}
}

func TestRingWriteKeepsTheCallersBuffers(t *testing.T) {
	r := newRing(2, 64, 0)

	rec := ringRecord("a", 64)
	rec.Payload = append(rec.Payload, "body"...)
	rec.Attempts = append(rec.Attempts, port.Attempt{Status: domain.DeliveryStatusSuccess})
	metadata := rec.Metadata
	r.write(&rec)

	// The slot owns the map the caller handed in, so a caller that kept it
	// would be writing into a stored record.
	metadata["k"] = "mutated"

	stored := r.page(0, 1).Records[0]
	if string(stored.Payload) != "body" || len(stored.Attempts) != 1 {
		t.Fatalf("stored payload, attempts = %q, %v, want the caller's buffers", stored.Payload, stored.Attempts)
	}

	if stored.Metadata["k"] != "mutated" {
		t.Fatalf("stored Metadata = %v, want the caller's map by reference", stored.Metadata)
	}

	if stored.PayloadSize != 4 {
		t.Fatalf("PayloadSize = %d, want 4", stored.PayloadSize)
	}
}

func TestRingWriteKeepsAPayloadAtTheLimit(t *testing.T) {
	r := newRing(1, 8, 0)

	rec := ringRecord("a", 8)
	rec.Payload = append(rec.Payload, "01234567"...)
	payload := rec.Payload
	r.write(&rec)

	stored := r.slots[0]
	if !samePayload(stored.Payload, payload) {
		t.Fatal("a payload of exactly the limit was copied, want the caller's buffer kept as is")
	}

	if stored.PayloadSize != 8 {
		t.Fatalf("PayloadSize = %d, want 8", stored.PayloadSize)
	}
}

func TestRingWriteTruncatesAnOversizedPayload(t *testing.T) {
	r := newRing(1, 8, 0)

	primer := ringRecord("primer", 8)
	r.write(&primer)

	rec := ringRecord("a", 64)
	rec.Payload = append(rec.Payload, "0123456789abcdef"...)
	oversized := rec.Payload
	r.write(&rec)

	stored := r.slots[0]
	if string(stored.Payload) != "01234567" || len(stored.Payload) != 8 || cap(stored.Payload) != 8 {
		t.Fatalf("stored payload = %q with cap %d, want %q with cap 8", stored.Payload, cap(stored.Payload), "01234567")
	}

	if stored.PayloadSize != 16 {
		t.Fatalf("PayloadSize = %d, want 16, the untruncated size", stored.PayloadSize)
	}

	if samePayload(stored.Payload, oversized) {
		t.Fatal("the slot kept the oversized buffer, want it dropped")
	}

	if samePayload(rec.Payload, oversized) || cap(rec.Payload) != 8 {
		t.Fatalf("caller got a %d byte buffer back, want the slot's previous one rather than its own oversized one", cap(rec.Payload))
	}
}

func TestRingPageReadsAcrossTheWrapSeam(t *testing.T) {
	r := newRing(4, 8, 0)

	for _, target := range []string{"a", "b", "c", "d", "e", "f"} {
		rec := ringRecord(target, 8)
		r.write(&rec)
	}

	if r.next != 2 {
		t.Fatalf("next = %d, want 2, with the newest records before the seam", r.next)
	}

	if got := targets(r.page(0, 4)); !equal(got, []string{"f", "e", "d", "c"}) {
		t.Fatalf("targets = %v, want [f e d c]", got)
	}

	if got := targets(r.page(2, 4)); !equal(got, []string{"d", "c"}) {
		t.Fatalf("targets = %v, want [d c], the window past the seam", got)
	}
}

func TestRingPageClampsOffsets(t *testing.T) {
	r := newRing(4, 8, 0)

	for _, target := range []string{"a", "b"} {
		rec := ringRecord(target, 8)
		r.write(&rec)
	}

	page := r.page(-5, 100)
	if page.Start != 0 || page.End != 2 || page.Total != 2 || len(page.Records) != 2 {
		t.Fatalf("page(-5, 100) = %+v, want both records clamped to [0, 2)", page)
	}

	page = r.page(5, 10)
	if page.Records != nil || page.Start != 2 || page.End != 2 || page.Total != 2 || page.Written != 2 {
		t.Fatalf("page(5, 10) = %+v, want an empty window with honest totals", page)
	}

	page = r.page(2, 1)
	if page.Records != nil || page.Start != 2 || page.End != 1 {
		t.Fatalf("page(2, 1) = %+v, want an empty window for an inverted range", page)
	}
}

func TestRingPageReportsWrittenPastCapacity(t *testing.T) {
	r := newRing(2, 8, 0)

	for _, target := range []string{"a", "b", "c", "d", "e"} {
		rec := ringRecord(target, 8)
		r.write(&rec)
	}

	page := r.page(0, 2)
	if page.Total != 2 || page.Written != 5 {
		t.Fatalf("Total, Written = %d, %d, want 2, 5", page.Total, page.Written)
	}
}

func TestRingPageReturnsCopies(t *testing.T) {
	r := newRing(2, 64, 0)

	rec := ringRecord("a", 64)
	rec.Payload = append(rec.Payload, "body"...)
	rec.Attempts = append(rec.Attempts, port.Attempt{Status: domain.DeliveryStatusSuccess})
	r.write(&rec)

	page := r.page(0, 1)
	page.Records[0].Payload[0] = 'X'
	page.Records[0].Attempts[0].Status = domain.DeliveryStatusFailure
	page.Records[0].Metadata["k"] = "mutated"

	stored := r.slots[0]
	if stored.Payload[0] == 'X' || stored.Attempts[0].Status != domain.DeliveryStatusSuccess || stored.Metadata["k"] == "mutated" {
		t.Fatal("page handed out the ring's own buffers")
	}
}

func TestRingSearchWithoutFiltersReadsEverything(t *testing.T) {
	r := searchRing()

	page := r.search(port.Query{End: 10})

	if got := targets(page); !equal(got, []string{"delta", "charlie", "bravo", "alpha"}) {
		t.Fatalf("targets = %v, want [delta charlie bravo alpha]", got)
	}

	if page.Total != 4 {
		t.Fatalf("Total = %d, want 4, every record in the ring", page.Total)
	}
}

func TestRingSearchFiltersByField(t *testing.T) {
	r := searchRing()

	cases := map[string]struct {
		query port.Query
		want  []string
	}{
		"sink type":        {port.Query{SinkTypes: []string{"http"}}, []string{"charlie"}},
		"sink types":       {port.Query{SinkTypes: []string{"http", "nats"}}, []string{"charlie", "bravo"}},
		"host":             {port.Query{Hosts: []string{"kafka-2"}}, []string{"delta"}},
		"hosts":            {port.Query{Hosts: []string{"kafka-1", "nats-1"}}, []string{"bravo", "alpha"}},
		"target key":       {port.Query{Targets: []string{"orders"}}, []string{"delta", "bravo"}},
		"rendered target":  {port.Query{Targets: []string{"alpha"}}, nil},
		"table":            {port.Query{Tables: []string{"public.orders"}}, []string{"delta", "bravo"}},
		"tables":           {port.Query{Tables: []string{"public.orders", "billing.users"}}, []string{"delta", "charlie", "bravo"}},
		"table prefix":     {port.Query{Tables: []string{"public.order"}}, nil},
		"table other half": {port.Query{Tables: []string{"billing.orders", "publicorders"}}, nil},
		"operations":       {port.Query{Operations: []domain.OperationType{domain.OperationTypeUpdate, domain.OperationTypeDelete}}, []string{"charlie", "bravo"}},
		"status":           {port.Query{Statuses: []domain.DeliveryStatus{domain.DeliveryStatusFailure}}, []string{"delta"}},
		"statuses":         {port.Query{Statuses: []domain.DeliveryStatus{domain.DeliveryStatusFailure, domain.DeliveryStatusSuccess}}, []string{"delta", "charlie", "bravo", "alpha"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.query.End = 10

			page := r.search(tc.query)
			if got := targets(page); !equal(got, tc.want) {
				t.Fatalf("targets = %v, want %v", got, tc.want)
			}

			if page.Total != len(tc.want) {
				t.Fatalf("Total = %d, want %d", page.Total, len(tc.want))
			}
		})
	}
}

func TestRingSearchCombinesFilters(t *testing.T) {
	r := searchRing()

	page := r.search(port.Query{Tables: []string{"public.orders"}, Statuses: []domain.DeliveryStatus{domain.DeliveryStatusSuccess}, End: 10})

	if got := targets(page); !equal(got, []string{"bravo"}) {
		t.Fatalf("targets = %v, want [bravo]", got)
	}
}

func TestRingSearchFindsTextWhereverItSits(t *testing.T) {
	r := searchRing()

	cases := map[string]struct {
		text string
		want []string
	}{
		"payload":        {"ORDER-7", []string{"bravo"}},
		"target":         {"CHARLIE", []string{"charlie"}},
		"record error":   {"refused", []string{"delta"}},
		"attempt error":  {"TIMED out", []string{"delta"}},
		"metadata key":   {"header.x-trace", []string{"alpha"}},
		"metadata value": {"post", []string{"charlie"}},
		"nothing":        {"absent", nil},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := targets(r.search(port.Query{Text: tc.text, End: 10})); !equal(got, tc.want) {
				t.Fatalf("targets = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRingSearchPagesTheMatchesRatherThanTheRing(t *testing.T) {
	r := searchRing()

	page := r.search(port.Query{Tables: []string{"public.orders"}, Start: 1, End: 2})

	// Offset 1 is the second match, not the second record: bravo rather than charlie.
	if got := targets(page); !equal(got, []string{"bravo"}) {
		t.Fatalf("targets = %v, want [bravo]", got)
	}

	if page.Total != 2 || page.Start != 1 || page.End != 2 {
		t.Fatalf("Total, Start, End = %d, %d, %d, want 2, 1, 2", page.Total, page.Start, page.End)
	}
}

func TestRingSearchClampsOffsetsToTheMatches(t *testing.T) {
	r := searchRing()

	page := r.search(port.Query{Statuses: []domain.DeliveryStatus{domain.DeliveryStatusFailure}, Start: 5, End: 10})

	if len(page.Records) != 0 {
		t.Fatalf("Records = %v, want none past the last match", targets(page))
	}

	if page.Total != 1 || page.Start != 1 || page.End != 1 {
		t.Fatalf("Total, Start, End = %d, %d, %d, want 1, 1, 1", page.Total, page.Start, page.End)
	}
}

func TestRingIndexCountsBackFromTheNewest(t *testing.T) {
	r := newRing(4, 8, 0)
	r.next = 1

	tests := []struct {
		offset int
		want   int
	}{
		{offset: 0, want: 0},
		{offset: 1, want: 3},
		{offset: 3, want: 1},
	}

	for _, tt := range tests {
		if got := r.index(tt.offset); got != tt.want {
			t.Fatalf("index(%d) = %d, want %d", tt.offset, got, tt.want)
		}
	}

	r.next = 0
	if got := r.index(0); got != 3 {
		t.Fatalf("index(0) = %d with next at 0, want the last slot", got)
	}
}

func TestCloneDeepCopiesBuffers(t *testing.T) {
	rec := ringRecord("a", 8)
	rec.Payload = append(rec.Payload, "body"...)
	rec.Attempts = append(rec.Attempts, port.Attempt{Status: domain.DeliveryStatusSuccess})

	out := clone(&rec)
	out.Payload[0] = 'X'
	out.Attempts[0].Status = domain.DeliveryStatusFailure
	out.Metadata["k"] = "mutated"

	if rec.Payload[0] == 'X' || rec.Attempts[0].Status != domain.DeliveryStatusSuccess || rec.Metadata["k"] != "a" {
		t.Fatal("clone shares buffers with the record it copied")
	}

	if out.Target != rec.Target || out.PayloadSize != rec.PayloadSize {
		t.Fatalf("clone = %+v, want the scalars of %+v", out, rec)
	}
}

func TestCloneKeepsAbsentMetadataNil(t *testing.T) {
	rec := port.Record{Target: "a"}

	if out := clone(&rec); out.Metadata != nil {
		t.Fatalf("Metadata = %v, want nil", out.Metadata)
	}
}

func TestClampBoundsValue(t *testing.T) {
	tests := []struct {
		v, max, want int
	}{
		{v: -1, max: 5, want: 0},
		{v: 0, max: 5, want: 0},
		{v: 3, max: 5, want: 3},
		{v: 5, max: 5, want: 5},
		{v: 9, max: 5, want: 5},
		{v: 3, max: 0, want: 0},
	}

	for _, tt := range tests {
		if got := clamp(tt.v, tt.max); got != tt.want {
			t.Fatalf("clamp(%d, %d) = %d, want %d", tt.v, tt.max, got, tt.want)
		}
	}
}

func TestContainsFold(t *testing.T) {
	cases := []struct {
		name   string
		hay    string
		needle string
		want   bool
	}{
		{name: "empty needle", hay: "anything", needle: "", want: true},
		{name: "empty needle in empty hay", hay: "", needle: "", want: true},
		{name: "case-insensitive match", hay: "Connection REFUSED", needle: "refused", want: true},
		{name: "absent", hay: "timeout", needle: "refused", want: false},
		{name: "needle longer than hay", hay: "ab", needle: "abc", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsFold(tc.hay, tc.needle); got != tc.want {
				t.Errorf("containsFold(%q, %q) = %v, want %v", tc.hay, tc.needle, got, tc.want)
			}

			if got := containsFold([]byte(tc.hay), tc.needle); got != tc.want {
				t.Errorf("containsFold([]byte(%q), %q) = %v, want %v", tc.hay, tc.needle, got, tc.want)
			}
		})
	}
}

func TestRingWriteDoesNotAllocateInSteadyState(t *testing.T) {
	r := newRing(8, 64, 0)
	rec := ringRecord("a", 64)

	// Until every slot has been written once the buffers handed back are the
	// zero value's nil ones, and the refill below allocates.
	for range len(r.slots) {
		refillRingRecord(&rec)
		r.write(&rec)
	}

	got := testing.AllocsPerRun(1000, func() {
		refillRingRecord(&rec)
		r.write(&rec)
	})

	if got != 0 {
		t.Fatalf("write allocated %v times per call, want 0", got)
	}
}

func TestRingWriteAllocatesOnceForAnOversizedPayload(t *testing.T) {
	r := newRing(8, 8, 0)
	rec := ringRecord("a", 64)
	// A payload the caller owns for good, so the refill of the buffer the ring
	// hands back cannot add an allocation of its own to the count.
	payload := []byte("0123456789abcdef")

	for range len(r.slots) {
		rec.Payload = payload
		r.write(&rec)
	}

	got := testing.AllocsPerRun(1000, func() {
		rec.Payload = payload
		r.write(&rec)
	})

	// The truncating copy, and nothing else: the caller's oversized buffer is
	// dropped, so every lap allocates one and only one payload.
	if got != 1 {
		t.Fatalf("write allocated %v times per call, want 1, the truncating copy", got)
	}
}

func TestRingPageDoesNotAllocateForAnEmptyWindow(t *testing.T) {
	r := newRing(4, 64, 0)
	rec := ringRecord("a", 64)
	r.write(&rec)

	got := testing.AllocsPerRun(1000, func() { r.page(5, 10) })

	if got != 0 {
		t.Fatalf("page allocated %v times per call for an empty window, want 0", got)
	}
}

func TestRingSearchDoesNotAllocateWhenNothingMatches(t *testing.T) {
	r := searchRing()

	got := testing.AllocsPerRun(1000, func() { r.search(port.Query{Text: "absent", End: 10}) })

	if got != 0 {
		t.Fatalf("search allocated %v times per call for an empty result, want 0", got)
	}
}

func TestRingSearchDoesNotAllocateWithListFilters(t *testing.T) {
	r := searchRing()
	q := port.Query{
		SinkTypes: []string{"kafka", "nats"},
		Hosts:     []string{"kafka-2", "nats-1"},
		Targets:   []string{"orders"},
		Tables:    []string{"billing.orders", "public.orders"},
		Statuses:  []domain.DeliveryStatus{domain.DeliveryStatusFailure},
		Text:      "absent",
		End:       10,
	}

	got := testing.AllocsPerRun(1000, func() { r.search(q) })

	if got != 0 {
		t.Fatalf("search allocated %v times per call for an empty result, want 0", got)
	}
}

// searchEpoch is the start time of the oldest record searchRing holds; the
// others follow one minute apart, so a reader can tell them apart by time.
var searchEpoch = time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)

// searchRing returns a full ring whose four records differ in every field a
// query can filter on, newest first: delta, charlie, bravo, alpha.
func searchRing() *ring {
	r := newRing(4, 64, 0)

	for _, rec := range []port.Record{
		{
			StartedAt: searchEpoch,
			Table:     domain.TableIdentifier{Schema: "public", Name: "users"},
			Operation: domain.OperationTypeInsert,
			SinkType:  "kafka",
			Host:      "kafka-1",
			Target:    "alpha",
			TargetKey: "users",
			Status:    domain.DeliveryStatusSuccess,
			Payload:   []byte(`{"id":1}`),
			Metadata:  map[string]string{"header.x-trace": "1"},
		},
		{
			StartedAt: searchEpoch.Add(time.Minute),
			Table:     domain.TableIdentifier{Schema: "public", Name: "orders"},
			Operation: domain.OperationTypeUpdate,
			SinkType:  "nats",
			Host:      "nats-1",
			Target:    "bravo",
			TargetKey: "orders",
			Status:    domain.DeliveryStatusSuccess,
			Payload:   []byte(`{"ref":"order-7"}`),
		},
		{
			StartedAt: searchEpoch.Add(2 * time.Minute),
			Table:     domain.TableIdentifier{Schema: "billing", Name: "users"},
			Operation: domain.OperationTypeDelete,
			SinkType:  "http",
			Host:      "hook.local",
			Target:    "charlie",
			TargetKey: "POST",
			Status:    domain.DeliveryStatusSuccess,
			Metadata:  map[string]string{"method": "POST"},
		},
		{
			StartedAt: searchEpoch.Add(3 * time.Minute),
			Table:     domain.TableIdentifier{Schema: "public", Name: "orders"},
			Operation: domain.OperationTypeInsert,
			SinkType:  "kafka",
			Host:      "kafka-2",
			Target:    "delta",
			TargetKey: "orders",
			Status:    domain.DeliveryStatusFailure,
			Error:     "nats: connection refused",
			Attempts:  []port.Attempt{{Status: domain.DeliveryStatusRetry, Error: "nats: timed out"}},
		},
	} {
		r.write(&rec)
	}

	return r
}

// ringRecord returns a record identified by target, with buffers of payloadCap
// the ring can hand back to a later caller.
func ringRecord(target string, payloadCap int) port.Record {
	return port.Record{
		Table:     domain.TableIdentifier{Schema: "public", Name: "users"},
		Operation: domain.OperationTypeInsert,
		SinkType:  "kafka",
		Target:    target,
		Status:    domain.DeliveryStatusSuccess,
		Payload:   make([]byte, 0, payloadCap),
		Metadata:  map[string]string{"k": target},
		Attempts:  make([]port.Attempt, 0, 4),
	}
}

// refillRingRecord refills the buffers a write handed back, the way a delivery
// would. It touches no clock, so a caller measuring the ring measures the ring.
func refillRingRecord(rec *port.Record) {
	rec.Payload = append(rec.Payload[:0], `{"id":1}`...)
	rec.Attempts = append(rec.Attempts[:0], port.Attempt{Status: domain.DeliveryStatusSuccess})

	if rec.Metadata == nil {
		rec.Metadata = make(map[string]string, 1)
	}

	rec.Metadata["k"] = "v"
}

// samePayload reports whether two payloads share a backing array.
func samePayload(a, b []byte) bool {
	if cap(a) == 0 || cap(b) == 0 {
		return false
	}

	return &a[:1][0] == &b[:1][0]
}

// sameAttempts reports whether two attempt slices share a backing array.
func sameAttempts(a, b []port.Attempt) bool {
	if cap(a) == 0 || cap(b) == 0 {
		return false
	}

	return &a[:1][0] == &b[:1][0]
}

// sameMetadata reports whether two maps are the same map.
func sameMetadata(a, b map[string]string) bool {
	if a == nil || b == nil {
		return false
	}

	return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
}

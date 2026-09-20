package process_operation

import (
	"context"
	"errors"
	"testing"
	"time"

	cfgresilience "hump/config/resilience"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	operationscollector "hump/internal/port/operations_collector"
	"hump/internal/port/sink"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func TestBindingSkipsOperationLogWhileDisconnected(t *testing.T) {
	log := &fakeOperationLog{}
	b, sinks := newTestBinding(t, log, 1)

	deliver(t, b)

	if len(log.records) != 0 {
		t.Fatalf("logged %d records, want none while disconnected", len(log.records))
	}

	if sinks.captured {
		t.Fatal("the sink was handed a capture although nobody is watching")
	}
}

func TestBindingLogsOneRecordPerDelivery(t *testing.T) {
	log := &fakeOperationLog{connected: true}
	b, _ := newTestBinding(t, log, 3, errBoom, errBoom)

	deliver(t, b)

	if len(log.records) != 1 {
		t.Fatalf("logged %d records, want exactly one for a delivery with two retries", len(log.records))
	}

	rec := log.records[0]
	if rec.Status != domain.DeliveryStatusSuccess || rec.Retries != 2 || len(rec.Attempts) != 3 {
		t.Fatalf("record = %q with %d retries and %d attempts, want success, 2, 3", rec.Status, rec.Retries, len(rec.Attempts))
	}
}

func TestOperationStatuses(t *testing.T) {
	tests := []struct {
		name     string
		attempts int
		errs     []error
		want     domain.DeliveryStatus
		wantEach []domain.DeliveryStatus
	}{
		{
			name: "delivered first try", attempts: 3,
			want:     domain.DeliveryStatusSuccess,
			wantEach: []domain.DeliveryStatus{domain.DeliveryStatusSuccess},
		},
		{
			name: "delivered after two retries", attempts: 3, errs: []error{errBoom, errBoom},
			want: domain.DeliveryStatusSuccess,
			wantEach: []domain.DeliveryStatus{
				domain.DeliveryStatusRetry, domain.DeliveryStatusRetry, domain.DeliveryStatusSuccess,
			},
		},
		{
			name: "attempts exhausted", attempts: 3, errs: []error{errBoom, errBoom, errBoom},
			want: domain.DeliveryStatusFailure,
			wantEach: []domain.DeliveryStatus{
				domain.DeliveryStatusRetry, domain.DeliveryStatusRetry, domain.DeliveryStatusFailure,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &fakeOperationLog{connected: true}
			b, _ := newTestBinding(t, log, tt.attempts, tt.errs...)

			_ = b.Process(context.Background(), newTestContext("a"))

			rec := log.records[0]
			if rec.Status != tt.want {
				t.Fatalf("Status = %q, want %q", rec.Status, tt.want)
			}

			if len(rec.Attempts) != len(tt.wantEach) {
				t.Fatalf("got %d attempts, want %d", len(rec.Attempts), len(tt.wantEach))
			}

			for i, want := range tt.wantEach {
				if rec.Attempts[i].Status != want {
					t.Fatalf("attempt %d = %q, want %q", i, rec.Attempts[i].Status, want)
				}
			}
		})
	}
}

func TestCaptureFilledOnFailedDelivery(t *testing.T) {
	log := &fakeOperationLog{connected: true}
	b, _ := newTestBinding(t, log, 1, errBoom)

	deliver(t, b)

	rec := log.records[0]
	if rec.Status != domain.DeliveryStatusFailure {
		t.Fatalf("Status = %q, want failure", rec.Status)
	}

	if string(rec.Payload) != "payload-a" || rec.Target != "target-a" || rec.Metadata["seq"] != "1" {
		t.Fatalf("record = %q, %q, %v, want the failed attempt's request", rec.Payload, rec.Target, rec.Metadata)
	}

	if rec.Attempts[0].Error == "" {
		t.Fatal("the failed attempt carries no error")
	}
}

func TestRecordCarriesTheTerminalError(t *testing.T) {
	tests := map[string]struct {
		errs      []error
		wantError bool
	}{
		"attempts exhausted":         {[]error{errBoom, errBoom}, true},
		"delivered after one retry":  {[]error{errBoom}, false},
		"delivered on the first try": {nil, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			log := &fakeOperationLog{connected: true}
			b, _ := newTestBinding(t, log, 2, tt.errs...)

			deliver(t, b)

			if got := log.records[0].Error; (got != "") != tt.wantError {
				t.Fatalf("Error = %q, want a terminal error: %v", got, tt.wantError)
			}
		})
	}
}

func TestBindingRebindsCaptureAfterRecord(t *testing.T) {
	log := &fakeOperationLog{connected: true}
	b, _ := newTestBinding(t, log, 1)

	if err := b.Process(context.Background(), newTestContext("a")); err != nil {
		t.Fatalf("Process() error = %v, want nil", err)
	}

	if err := b.Process(context.Background(), newTestContext("b")); err != nil {
		t.Fatalf("Process() error = %v, want nil", err)
	}

	if got := string(log.records[0].Payload); got != "payload-a" {
		t.Fatalf("first record = %q, want %q: the second delivery wrote through the reused buffer", got, "payload-a")
	}

	if got := string(log.records[1].Payload); got != "payload-b" {
		t.Fatalf("second record = %q, want %q", got, "payload-b")
	}
}

func TestBindingReusesBuffersAcrossDeliveries(t *testing.T) {
	log := &fakeOperationLog{connected: true, recycle: true}
	b, _ := newTestBinding(t, log, 1)

	// The first deliveries swap in the log's empty buffers, the way the ring's
	// untouched slots do; steady state starts once they have been grown.
	deliver(t, b)
	deliver(t, b)
	first := cap(b.op.capture.Payload)

	for range 100 {
		deliver(t, b)
	}

	if cap(b.op.capture.Payload) != first {
		t.Fatalf("payload buffer grew from %d to %d, want it reused", first, cap(b.op.capture.Payload))
	}
}

func TestBindingKeepsCountersWhileLogging(t *testing.T) {
	log := &fakeOperationLog{connected: true}
	observer := &fakeObserver{}
	b, _ := newTestBinding(t, log, 3, errBoom, errBoom)
	b.observer = observer

	deliver(t, b)

	// One retry counter per repeated attempt, still reported alongside the
	// single log record that holds those retries.
	if got := observer.calls[b.conn][domain.DeliveryStatusRetry]; got != 2 {
		t.Fatalf("retry count = %d, want 2", got)
	}

	if got := observer.calls[b.conn][domain.DeliveryStatusSuccess]; got != 1 {
		t.Fatalf("success count = %d, want 1", got)
	}
}

// BenchmarkBindingProcessWithoutLog is the control for BenchmarkBindingProcessIdle:
// a binding built before the operations log existed. The two must match, since
// a disconnected log takes the same chain. Neither is allocation free — the
// per-attempt context.WithTimeout was always there.
func BenchmarkBindingProcessWithoutLog(b *testing.B) {
	binding, _ := newTestBinding(b, nil, 1)
	operationContext := newTestContext("a")
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = binding.Process(ctx, operationContext)
	}
}

// BenchmarkBindingProcessLogged measures the connected path. It is not
// allocation free and cannot be: building the Metadata entries concatenates a
// prefix per header, which is the price of the single map the record carries.
// The ring itself still adds nothing — see BenchmarkRecordActive.
func BenchmarkBindingProcessLogged(b *testing.B) {
	log := &benchOperationLog{}
	binding, _ := newTestBinding(b, log, 1)
	operationContext := newTestContext("a")
	ctx := context.Background()

	// Two laps to grow the swapped buffers, as the ring's first lap does.
	_ = binding.Process(ctx, operationContext)
	_ = binding.Process(ctx, operationContext)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = binding.Process(ctx, operationContext)
	}
}

func BenchmarkBindingProcessIdle(b *testing.B) {
	log := &fakeOperationLog{}
	binding, _ := newTestBinding(b, log, 1)
	operationContext := newTestContext("a")
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = binding.Process(ctx, operationContext)
	}
}

// errBoom is the failure a fake sink returns for a scripted attempt.
var errBoom = errors.New("boom")

// newTestBinding returns a binding whose sink fails the first len(errs)
// attempts, together with that sink.
func newTestBinding(tb testing.TB, log OperationLog, attempts int, errs ...error) (*binding, *fakeSink) {
	tb.Helper()

	factory := promauto.With(prometheus.NewRegistry())
	counter := factory.NewCounterVec(prometheus.CounterOpts{Name: "sink_messages_total"}, []string{"status", "type", "host", "target"})
	histogram := factory.NewHistogramVec(prometheus.HistogramOpts{Name: "sink_duration_seconds"}, []string{"type", "host", "target"})

	sinks := &fakeSink{errs: errs}
	cfg := sinkcfg.Config{
		Sink:    fakeDefinition{},
		Failure: &cfgresilience.Failure{},
		Retry:   &cfgresilience.Retry{MaxAttempts: attempts, AttemptTimeout: time.Second},
	}
	cfg.Failure.SetDefaults()

	return newBinding(cfg, "public", "users", sinks, counter, histogram, nil, log), sinks
}

// newTestContext returns the operation context of one row, identified by id.
func newTestContext(id string) *sink.SinkContext {
	return &sink.SinkContext{
		Timestamp: time.Now(),
		Operation: domain.OperationTypeUpdate,
		Database:  "app",
		Schema:    "public",
		Table:     "users",
		Next:      domain.DbRecord{"id": id},
	}
}

// deliver runs one delivery of row "a" and fails the test on an unexpected error.
func deliver(tb testing.TB, b *binding) {
	tb.Helper()

	if err := b.Process(context.Background(), newTestContext("a")); err != nil && !errors.Is(err, errBoom) {
		tb.Fatalf("Process() error = %v", err)
	}
}

// fakeSink fills the capture the way a real adapter does and returns the
// scripted error of each attempt, succeeding once the script runs out.
type fakeSink struct {
	errs     []error
	calls    int
	captured bool
}

// Process implements [sink.Sink].
func (s *fakeSink) Process(_ context.Context, _ sink.Definition, operationContext *sink.SinkContext, capture *sink.Capture) error {
	s.calls++

	if capture != nil {
		s.captured = true
		id, _ := operationContext.Next["id"].(string)
		capture.Target = "target-" + id
		capture.SetPayload([]byte("payload-" + id))
		capture.Set("seq", "1")
	}

	if s.calls <= len(s.errs) {
		return s.errs[s.calls-1]
	}

	return nil
}

// benchOperationLog swaps the buffers the way the ring does and keeps nothing,
// so a benchmark measures the delivery path rather than a fake's bookkeeping.
type benchOperationLog struct {
	payload  []byte
	attempts []operationscollector.Attempt
	metadata map[string]string
}

func (l *benchOperationLog) Connected() bool { return true }

// Record implements [OperationLog].
func (l *benchOperationLog) Record(rec *operationscollector.Record) {
	l.payload, rec.Payload = rec.Payload, l.payload[:0]
	l.attempts, rec.Attempts = rec.Attempts, l.attempts[:0]
	l.metadata, rec.Metadata = rec.Metadata, l.metadata
}

// fakeDefinition is a sink definition with a fixed host and target.
type fakeDefinition struct{}

func (fakeDefinition) Type() sink.SinkType                        { return sink.SinkTypeKafka }
func (fakeDefinition) DisplayName() string                        { return "fake" }
func (fakeDefinition) Metadata(sink.SinkContext) (string, string) { return "host", "configured" }

// fakeOperationLog keeps every record it is handed. It copies the buffers,
// since the real collector swaps them away from the caller. With recycle set it
// also hands buffers back, the way a ring with populated slots does.
type fakeOperationLog struct {
	connected bool
	recycle   bool
	records   []operationscollector.Record
	spare     []byte
}

func (l *fakeOperationLog) Connected() bool { return l.connected }

// Record implements [OperationLog].
func (l *fakeOperationLog) Record(rec *operationscollector.Record) {
	stored := *rec
	stored.Payload = append([]byte(nil), rec.Payload...)
	stored.Attempts = append([]operationscollector.Attempt(nil), rec.Attempts...)
	stored.Metadata = make(map[string]string, len(rec.Metadata))

	for k, v := range rec.Metadata {
		stored.Metadata[k] = v
	}

	l.records = append(l.records, stored)

	if l.recycle {
		rec.Payload, l.spare = l.spare[:0], rec.Payload[:0]
	} else {
		rec.Payload = rec.Payload[:0]
	}

	rec.Attempts = rec.Attempts[:0]
}

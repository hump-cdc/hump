package process_operation

import (
	"context"
	"testing"
	"time"

	cfgresilience "hump/config/resilience"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	metricscollector "hump/internal/port/metrics_collector"
	"hump/internal/port/sink"
	"hump/pkg/resilience"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func TestBindingSkipsRecordWithoutObserver(_ *testing.T) {
	// main passes a Collector that is nil when the web UI is disabled; converting
	// one nil interface to another has to keep it nil.
	var collector metricscollector.Collector
	b := &binding{observer: collector}

	b.record(domain.DeliveryStatusSuccess)
}

func TestBindingRecordsEveryOutcome(t *testing.T) {
	observer := &fakeObserver{}
	conn := metricscollector.Connection{
		Table:    domain.TableIdentifier{Schema: "public", Name: "users"},
		SinkType: "kafka",
		Host:     "h",
		Target:   "users",
	}
	b := &binding{observer: observer, conn: conn}

	b.record(domain.DeliveryStatusSuccess)
	b.record(domain.DeliveryStatusRetry)
	b.record(domain.DeliveryStatusFailure)

	want := map[domain.DeliveryStatus]int{
		domain.DeliveryStatusSuccess: 1,
		domain.DeliveryStatusRetry:   1,
		domain.DeliveryStatusFailure: 1,
	}
	for status, count := range want {
		if observer.calls[conn][status] != count {
			t.Fatalf("got %v for %q, want %d", observer.calls[conn][status], status, count)
		}
	}
}

func TestBindingAcceptsConfiguredOperations(t *testing.T) {
	b := &binding{operations: map[domain.OperationType]struct{}{
		domain.OperationTypeInsert: {},
		domain.OperationTypeDelete: {},
	}}

	for op, want := range map[domain.OperationType]bool{
		domain.OperationTypeInsert:   true,
		domain.OperationTypeUpdate:   false,
		domain.OperationTypeDelete:   true,
		domain.OperationTypeTruncate: false,
	} {
		if got := b.accepts(op); got != want {
			t.Fatalf("accepts(%q) = %v, want %v", op, got, want)
		}
	}
}

func TestBindingSkipModeSwallowsFailure(t *testing.T) {
	sinks := &fakeSink{errs: []error{errBoom}}
	b := newBinding(sinkcfg.Config{
		Sink:    fakeDefinition{},
		Failure: &cfgresilience.Failure{Mode: resilience.FailureModeSkip},
		Retry:   &cfgresilience.Retry{MaxAttempts: 1, AttemptTimeout: time.Second},
	}, "public", "users", sinks, newTestCounter(), newTestHistogram(), nil, nil)

	if err := b.Process(context.Background(), newTestContext("a")); err != nil {
		t.Fatalf("Process() error = %v, want nil in skip mode", err)
	}
	if sinks.calls != 1 {
		t.Fatalf("sink called %d times, want 1", sinks.calls)
	}
}

func TestBindingDeliversOverlappingDeliveryUnlogged(t *testing.T) {
	log := &fakeOperationLog{connected: true}
	sinks := &reentrantSink{}
	cfg := sinkcfg.Config{
		Sink:       fakeDefinition{},
		Failure:    &cfgresilience.Failure{},
		Retry:      &cfgresilience.Retry{MaxAttempts: 1, AttemptTimeout: time.Second},
		Operations: []domain.OperationType{domain.OperationTypeUpdate},
	}
	cfg.Failure.SetDefaults()
	sinks.binding = newBinding(cfg, "public", "users", sinks, newTestCounter(), newTestHistogram(), nil, log)

	if !sinks.binding.accepts(domain.OperationTypeUpdate) || sinks.binding.accepts(domain.OperationTypeInsert) {
		t.Fatalf("operations = %v, want only update", sinks.binding.operations)
	}

	if err := sinks.binding.Process(context.Background(), newTestContext("a")); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(sinks.captures) != 2 || sinks.captures[0] == nil || sinks.captures[1] != nil {
		t.Fatalf("captures = %v, want the outer delivery logged and the nested one plain", sinks.captures)
	}
	if len(log.records) != 1 {
		t.Fatalf("logged %d records, want 1", len(log.records))
	}
}

// fakeObserver collects what a binding reported, by connection and status.
type fakeObserver struct {
	calls map[metricscollector.Connection]map[domain.DeliveryStatus]int
}

// Record implements [Observer].
func (o *fakeObserver) Record(conn metricscollector.Connection, status domain.DeliveryStatus, n int) {
	if o.calls == nil {
		o.calls = make(map[metricscollector.Connection]map[domain.DeliveryStatus]int)
	}

	if o.calls[conn] == nil {
		o.calls[conn] = make(map[domain.DeliveryStatus]int)
	}

	o.calls[conn][status] += n
}

// reentrantSink delivers through binding once more from inside its first
// delivery, the way a second caller overlapping the first one would, and
// records the capture each call was handed.
type reentrantSink struct {
	binding  *binding
	captures []*sink.Capture
}

// Process implements [sink.Sink].
func (s *reentrantSink) Process(ctx context.Context, _ sink.Definition, operationContext *sink.Context, capture *sink.Capture) error {
	s.captures = append(s.captures, capture)

	if len(s.captures) == 1 {
		return s.binding.Process(ctx, operationContext)
	}

	return nil
}

// newTestCounter returns a sink messages counter on a private registry.
func newTestCounter() *prometheus.CounterVec {
	return promauto.With(prometheus.NewRegistry()).
		NewCounterVec(prometheus.CounterOpts{Name: "sink_messages_total"}, []string{"status", "type", "host", "target"})
}

// newTestHistogram returns a sink duration histogram on a private registry.
func newTestHistogram() *prometheus.HistogramVec {
	return promauto.With(prometheus.NewRegistry()).
		NewHistogramVec(prometheus.HistogramOpts{Name: "sink_duration_seconds"}, []string{"type", "host", "target"})
}

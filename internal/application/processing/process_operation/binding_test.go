package process_operation

import (
	"testing"

	"hump/internal/domain"
	metricscollector "hump/internal/port/metrics_collector"
)

func TestBindingSkipsRecordWithoutObserver(t *testing.T) {
	// main passes a Collector that is nil when telemetry is disabled; converting
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

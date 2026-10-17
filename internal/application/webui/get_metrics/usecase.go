package getmetrics

import (
	"errors"
	"time"

	metricscollector "hump/internal/port/metrics_collector"
)

// ErrDisabled indicates that telemetry is off, so there are no metrics to serve.
var ErrDisabled = errors.New("telemetry disabled")

// Metrics is one snapshot of the collector aggregated into buckets.
type Metrics struct {
	Duration metricscollector.Duration
	Bucket   time.Duration // Bucket is the width of one aggregated point.
	Snapshot metricscollector.Snapshot
}

// UseCase dumps the metrics collector at a chosen bucket width.
type UseCase struct {
	collector metricscollector.Collector // collector is nil when telemetry is disabled.
}

// NewUseCase returns a use case serving the snapshots of collector.
func NewUseCase(collector metricscollector.Collector) *UseCase {
	return &UseCase{
		collector: collector,
	}
}

// Metrics returns the newest snapshot aggregated by duration. It fails with
// ErrDisabled without a collector, and with the collector's error for a
// duration it does not support.
func (uc *UseCase) Metrics(duration string) (Metrics, error) {
	if uc.collector == nil {
		return Metrics{}, ErrDisabled
	}

	d := metricscollector.Duration(duration)
	bucket, err := d.Duration()
	if err != nil {
		return Metrics{}, err
	}

	snapshot, err := uc.collector.Dump(d)
	if err != nil {
		return Metrics{}, err
	}

	return Metrics{Duration: d, Bucket: bucket, Snapshot: snapshot}, nil
}

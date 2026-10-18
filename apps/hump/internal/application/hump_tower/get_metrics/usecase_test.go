package getmetrics

import (
	"errors"
	"testing"
	"time"

	"hump/internal/domain"
	metricscollector "hump/internal/port/metrics_collector"
)

func TestMetricsFailsWithoutCollector(t *testing.T) {
	if _, err := NewUseCase(nil).Metrics("1m"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Metrics() error = %v, want ErrDisabled", err)
	}
}

func TestMetricsRejectsAnUnknownDuration(t *testing.T) {
	collector := &fakeCollector{}

	if _, err := NewUseCase(collector).Metrics("7m"); err == nil || errors.Is(err, ErrDisabled) {
		t.Fatalf("Metrics() error = %v, want a duration error", err)
	}

	if collector.dumped != "" {
		t.Errorf("collector dumped %q, want no dump for an unknown duration", collector.dumped)
	}
}

func TestMetricsReportsADumpFailure(t *testing.T) {
	collector := &fakeCollector{err: errors.New("not enough points")}

	if _, err := NewUseCase(collector).Metrics("5m"); !errors.Is(err, collector.err) {
		t.Fatalf("Metrics() error = %v, want the collector's", err)
	}
}

func TestMetricsReturnsTheSnapshotWithItsBucket(t *testing.T) {
	snapshot := metricscollector.Snapshot{Points: []metricscollector.Point{{Delivered: 3}}}
	collector := &fakeCollector{snapshot: snapshot}

	got, err := NewUseCase(collector).Metrics("10m")
	if err != nil {
		t.Fatalf("Metrics() error = %v", err)
	}

	if got.Duration != metricscollector.Duration10m || got.Bucket != 10*time.Minute {
		t.Errorf("Duration, Bucket = %q, %s, want 10m buckets", got.Duration, got.Bucket)
	}

	if collector.dumped != metricscollector.Duration10m || len(got.Snapshot.Points) != 1 || got.Snapshot.Points[0].Delivered != 3 {
		t.Errorf("Snapshot = %+v after dumping %q, want the collector's 10m snapshot", got.Snapshot, collector.dumped)
	}
}

// fakeCollector answers Dump with a fixed snapshot or error and keeps the
// duration it was asked for. Only Dump is reachable from the use case.
type fakeCollector struct {
	snapshot metricscollector.Snapshot
	err      error
	dumped   metricscollector.Duration
}

func (c *fakeCollector) Connect()                                                       {}
func (c *fakeCollector) Disconnect()                                                    {}
func (c *fakeCollector) Scrape() <-chan struct{}                                        { return nil }
func (c *fakeCollector) Record(metricscollector.Connection, domain.DeliveryStatus, int) {}

func (c *fakeCollector) Dump(d metricscollector.Duration) (metricscollector.Snapshot, error) {
	c.dumped = d

	return c.snapshot, c.err
}

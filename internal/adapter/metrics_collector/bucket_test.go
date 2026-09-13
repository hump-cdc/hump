package metricscollector

import (
	"math"
	"testing"
	"time"
)

func TestSeriesAlignsScrapesToBucketStart(t *testing.T) {
	s := &series{duration: time.Minute}

	s.add(bucketBase.Add(5*time.Second), 10*time.Second, observedDeliveries(3, 1))
	s.add(bucketBase.Add(55*time.Second), 10*time.Second, observedDeliveries(2, 0))
	s.add(bucketBase.Add(65*time.Second), 10*time.Second, observedDeliveries(4, 0))

	points := s.dump()
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}

	if !points[0].Timestamp.Equal(bucketBase) || !points[1].Timestamp.Equal(bucketBase.Add(time.Minute)) {
		t.Fatalf("got timestamps %v, %v", points[0].Timestamp, points[1].Timestamp)
	}

	if points[0].Delivered != 5 || points[0].Failed != 1 || points[1].Delivered != 4 {
		t.Fatalf("got delivered %v/%v failed %v", points[0].Delivered, points[1].Delivered, points[0].Failed)
	}

	if points[0].Throughput != 0.25 {
		t.Fatalf("got throughput %v, want 0.25", points[0].Throughput)
	}

	if points[0].SinkDelivery["kafka"] != 6 {
		t.Fatalf("got sink delivery %v, want 6", points[0].SinkDelivery["kafka"])
	}
}

func TestSeriesKeepsNewestPoints(t *testing.T) {
	s := &series{duration: time.Minute}

	for i := range pointsMax + 5 {
		s.add(bucketBase.Add(time.Duration(i)*time.Minute), time.Second, observedDeliveries(float64(i), 0))
	}

	points := s.dump()
	if len(points) != pointsMax {
		t.Fatalf("got %d points, want %d", len(points), pointsMax)
	}

	if points[0].Delivered != 5 || points[pointsMax-1].Delivered != float64(pointsMax+4) {
		t.Fatalf("got oldest %v newest %v", points[0].Delivered, points[pointsMax-1].Delivered)
	}
}

func TestSeriesFoldsGaugesByMax(t *testing.T) {
	s := &series{duration: time.Minute}

	for _, lag := range []float64{10, 30, 20} {
		observed := newAggregate()
		observed.lag = lag
		observed.memory = lag * 2
		s.add(bucketBase, time.Second, observed)
	}

	point := s.dump()[0]
	if point.ReplicationLagBytes != 30 || point.MemoryAllocBytes != 60 {
		t.Fatalf("got lag %v memory %v, want 30 and 60", point.ReplicationLagBytes, point.MemoryAllocBytes)
	}
}

func TestHistogramQuantileInterpolates(t *testing.T) {
	h := histogram{buckets: map[float64]float64{0.1: 50, 0.5: 90, 1: 100}, count: 100}

	got, ok := h.quantile(0.95)
	if !ok || math.Abs(got-0.75) > 1e-9 {
		t.Fatalf("got %v, %v, want 0.75", got, ok)
	}
}

func TestHistogramQuantileInInfBucket(t *testing.T) {
	h := histogram{buckets: map[float64]float64{0.1: 50, 0.5: 60}, count: 100}

	if _, ok := h.quantile(0.95); ok {
		t.Fatal("got an estimate for a quantile in the +Inf bucket")
	}
}

func TestHistogramQuantileEmpty(t *testing.T) {
	h := histogram{buckets: map[float64]float64{0.1: 0}}

	if _, ok := h.quantile(0.95); ok {
		t.Fatal("got an estimate for an empty histogram")
	}
}

func TestHistogramQuantileInEmptyLowestBucket(t *testing.T) {
	h := histogram{buckets: map[float64]float64{0.1: 0, 1: 10}, count: 10}

	got, ok := h.quantile(0)
	if !ok || got != 0 {
		t.Fatalf("got %v, %v, want 0", got, ok)
	}
}

func TestPointOmitsLatencyWithoutEstimate(t *testing.T) {
	s := &series{duration: time.Minute}

	observed := newAggregate()
	observed.addLatency("kafka", histogram{buckets: map[float64]float64{0.1: 10}, count: 10})
	observed.addLatency("http", histogram{buckets: map[float64]float64{0.1: 0}})
	s.add(bucketBase, time.Second, observed)

	latency := s.dump()[0].SinkLatencyP95Seconds
	if _, ok := latency["http"]; ok || len(latency) != 1 {
		t.Fatalf("got latency %v, want kafka only", latency)
	}
}

// bucketBase is a timestamp aligned to every bucket width.
var bucketBase = time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)

// observedDeliveries returns a scrape of kafka deliveries.
func observedDeliveries(delivered, failed float64) aggregate {
	observed := newAggregate()
	observed.delivered = delivered
	observed.failed = failed
	observed.sinkDelivery["kafka"] = delivered + failed

	return observed
}

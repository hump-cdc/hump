package metricscollector

import (
	"math"
	"slices"
	"time"

	port "hump/internal/port/metrics_collector"
)

// latencyQuantile is the quantile reported as sink latency.
const latencyQuantile = 0.95

// aggregate holds what one scrape observed, or what several folded into one bucket.
type aggregate struct {
	delivered float64
	failed    float64
	// lag and memory are gauge readings, folded by max.
	lag    float64
	memory float64
	// sinkDelivery counts delivered and failed messages by sink type.
	sinkDelivery map[string]float64
	// sinkLatency is the latency distribution by sink type.
	sinkLatency map[string]*histogram
}

// newAggregate returns an empty aggregate with its maps allocated.
func newAggregate() aggregate {
	return aggregate{
		sinkDelivery: make(map[string]float64),
		sinkLatency:  make(map[string]*histogram),
	}
}

// merge folds other into a: counts and distributions add up, gauges keep the peak.
func (a *aggregate) merge(other aggregate) {
	a.delivered += other.delivered
	a.failed += other.failed
	a.lag = max(a.lag, other.lag)
	a.memory = max(a.memory, other.memory)

	for sinkType, count := range other.sinkDelivery {
		a.sinkDelivery[sinkType] += count
	}

	for sinkType, h := range other.sinkLatency {
		a.addLatency(sinkType, *h)
	}
}

// addLatency adds h to the distribution of sinkType.
func (a *aggregate) addLatency(sinkType string, h histogram) {
	existing, ok := a.sinkLatency[sinkType]
	if !ok {
		existing = &histogram{buckets: make(map[float64]float64)}
		a.sinkLatency[sinkType] = existing
	}

	existing.add(h)
}

// bucket is the aggregate of the scrapes whose time fell into [start, start+width).
type bucket struct {
	start time.Time
	// covered is the scrape time attributed to the bucket, the throughput divisor.
	covered time.Duration
	aggregate
}

// point renders b for the port.
func (b *bucket) point() port.Point {
	p := port.Point{
		Timestamp:             b.start,
		Delivered:             b.delivered,
		Failed:                b.failed,
		ReplicationLagBytes:   b.lag,
		MemoryAllocBytes:      b.memory,
		SinkDelivery:          make(map[string]float64, len(b.sinkDelivery)),
		SinkLatencyP95Seconds: make(map[string]float64, len(b.sinkLatency)),
	}

	if b.covered > 0 {
		p.Throughput = b.delivered / b.covered.Seconds()
	}

	for sinkType, count := range b.sinkDelivery {
		p.SinkDelivery[sinkType] = count
	}

	for sinkType, h := range b.sinkLatency {
		if value, ok := h.quantile(latencyQuantile); ok {
			p.SinkLatencyP95Seconds[sinkType] = value
		}
	}

	return p
}

// series keeps the newest pointsMax buckets of one width, oldest first.
type series struct {
	duration time.Duration
	buckets  []*bucket
}

// add folds a scrape taken at now, which covered elapsed, into its bucket,
// opening a new bucket and dropping the oldest one past pointsMax as needed.
func (s *series) add(now time.Time, elapsed time.Duration, observed aggregate) {
	start := now.Truncate(s.duration)

	if n := len(s.buckets); n == 0 || !s.buckets[n-1].start.Equal(start) {
		s.buckets = append(s.buckets, &bucket{start: start, aggregate: newAggregate()})

		if extra := len(s.buckets) - pointsMax; extra > 0 {
			s.buckets = slices.Delete(s.buckets, 0, extra)
		}
	}

	current := s.buckets[len(s.buckets)-1]
	current.covered += elapsed
	current.merge(observed)
}

// dump renders every bucket for the port, oldest first.
func (s *series) dump() []port.Point {
	points := make([]port.Point, 0, len(s.buckets))
	for _, b := range s.buckets {
		points = append(points, b.point())
	}

	return points
}

// histogram is a cumulative latency distribution.
type histogram struct {
	// buckets maps an upper bound to the observations at or below it.
	buckets map[float64]float64
	// count is every observation, the implicit +Inf bucket.
	count float64
}

// add adds other to h, bound by bound.
func (h *histogram) add(other histogram) {
	for bound, count := range other.buckets {
		h.buckets[bound] += count
	}
	h.count += other.count
}

// sub returns h minus base, bound by bound.
func (h histogram) sub(base histogram) histogram {
	diff := histogram{buckets: make(map[float64]float64, len(h.buckets)), count: h.count - base.count}
	for bound, count := range h.buckets {
		diff.buckets[bound] = count - base.buckets[bound]
	}

	return diff
}

// quantile estimates the value below which q of the observations fall, interpolating
// linearly within a bucket as Prometheus' histogram_quantile does. It reports false
// when h is empty or the quantile lands in the unbounded +Inf bucket.
func (h histogram) quantile(q float64) (float64, bool) {
	if h.count <= 0 {
		return 0, false
	}

	bounds := make([]float64, 0, len(h.buckets))
	for bound := range h.buckets {
		if !math.IsInf(bound, 1) {
			bounds = append(bounds, bound)
		}
	}
	slices.Sort(bounds)

	rank := q * h.count
	lowerBound, lowerCount := 0.0, 0.0

	for _, bound := range bounds {
		count := h.buckets[bound]
		if count >= rank {
			within := count - lowerCount
			if within <= 0 {
				return lowerBound, true
			}

			return lowerBound + (bound-lowerBound)*(rank-lowerCount)/within, true
		}

		lowerBound, lowerCount = bound, count
	}

	return 0, false
}

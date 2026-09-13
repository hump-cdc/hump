package metricscollector

import (
	"strings"

	dto "github.com/prometheus/client_model/go"
)

const (
	familySinkMessages   = "hump_sink_messages_total"
	familySinkDuration   = "hump_sink_duration_seconds"
	familyReplicationLag = "hump_replication_lag_bytes"
	familyMemoryAlloc    = "go_memstats_alloc_bytes"

	labelStatus = "status"
	labelType   = "type"

	statusSuccess = "success"
	statusFailure = "failure"
)

// scraper turns gathered metric families into the aggregate observed since the
// previous scrape, keeping the cumulative readings of every series as baselines.
type scraper struct {
	counters   map[string]float64
	histograms map[string]histogram
}

// newScraper returns a scraper with no baselines.
func newScraper() *scraper {
	return &scraper{
		counters:   make(map[string]float64),
		histograms: make(map[string]histogram),
	}
}

// scrape reads families into an aggregate. Counters and histograms contribute how
// far they advanced since the previous scrape; a series seen for the first time
// only records its baseline. Gauges contribute their current reading.
func (s *scraper) scrape(families []*dto.MetricFamily) aggregate {
	observed := newAggregate()

	for _, family := range families {
		name := family.GetName()

		switch name {
		case familySinkMessages:
			for _, m := range family.GetMetric() {
				status := labelValue(m, labelStatus)
				if status != statusSuccess && status != statusFailure {
					continue
				}

				delta, ok := s.counterDelta(seriesKey(name, m), m.GetCounter().GetValue())
				if !ok {
					continue
				}

				observed.sinkDelivery[labelValue(m, labelType)] += delta
				if status == statusSuccess {
					observed.delivered += delta
				} else {
					observed.failed += delta
				}
			}

		case familySinkDuration:
			for _, m := range family.GetMetric() {
				delta, ok := s.histogramDelta(seriesKey(name, m), histogramOf(m.GetHistogram()))
				if ok {
					observed.addLatency(labelValue(m, labelType), delta)
				}
			}

		case familyReplicationLag:
			for _, m := range family.GetMetric() {
				observed.lag = max(observed.lag, m.GetGauge().GetValue())
			}

		case familyMemoryAlloc:
			for _, m := range family.GetMetric() {
				observed.memory = max(observed.memory, m.GetGauge().GetValue())
			}
		}
	}

	return observed
}

// counterDelta returns how far the counter at key advanced to value, and false
// when key has no baseline yet. The counters belong to this process, so they
// never reset.
func (s *scraper) counterDelta(key string, value float64) (float64, bool) {
	baseline, ok := s.counters[key]
	s.counters[key] = value

	return value - baseline, ok
}

// histogramDelta returns how far the histogram at key advanced to value, and
// false when key has no baseline yet.
func (s *scraper) histogramDelta(key string, value histogram) (histogram, bool) {
	baseline, ok := s.histograms[key]
	s.histograms[key] = value

	return value.sub(baseline), ok
}

// histogramOf converts a gathered histogram into its cumulative distribution.
func histogramOf(h *dto.Histogram) histogram {
	result := histogram{
		buckets: make(map[float64]float64, len(h.GetBucket())),
		count:   float64(h.GetSampleCount()),
	}

	for _, b := range h.GetBucket() {
		result.buckets[b.GetUpperBound()] = float64(b.GetCumulativeCount())
	}

	return result
}

// seriesKey identifies one series of a family by its label pairs, which the
// registry gathers sorted by name.
func seriesKey(family string, m *dto.Metric) string {
	var b strings.Builder
	b.WriteString(family)

	for _, pair := range m.GetLabel() {
		b.WriteByte(0)
		b.WriteString(pair.GetName())
		b.WriteByte('=')
		b.WriteString(pair.GetValue())
	}

	return b.String()
}

// labelValue returns the value of label name on m, or "" when m has no such label.
func labelValue(m *dto.Metric, name string) string {
	for _, pair := range m.GetLabel() {
		if pair.GetName() == name {
			return pair.GetValue()
		}
	}

	return ""
}

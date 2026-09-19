package metricscollector

import (
	"strings"

	"hump/internal/domain"
	port "hump/internal/port/metrics_collector"

	dto "github.com/prometheus/client_model/go"
)

const (
	familySinkMessages   = "hump_sink_messages_total"
	familySinkDuration   = "hump_sink_duration_seconds"
	familyProcessedRows  = "hump_processed_rows_duration_seconds"
	familyReplicationLag = "hump_replication_lag_bytes"
	familyMemoryAlloc    = "go_memstats_alloc_bytes"

	labelStatus    = "status"
	labelType      = "type"
	labelHost      = "host"
	labelTarget    = "target"
	labelOperation = "operation"
	labelSchema    = "schema"
	labelTable     = "table"
)

// sinkKey identifies one rendered destination of one sink endpoint, matching the
// label set the pipeline writes its sink metrics with.
type sinkKey struct {
	sinkType string
	host     string
	target   string
}

// latest holds the absolute counter readings of the most recent scrape, which is
// what the flat views of a snapshot report.
type latest struct {
	tables map[domain.TableIdentifier]port.TableOps
	sinks  map[sinkKey]map[domain.DeliveryStatus]float64
}

// newLatest returns an empty latest with its maps allocated.
func newLatest() latest {
	return latest{
		tables: make(map[domain.TableIdentifier]port.TableOps),
		sinks:  make(map[sinkKey]map[domain.DeliveryStatus]float64),
	}
}

// scraper turns gathered metric families into the aggregate observed since the
// previous scrape, keeping the cumulative readings of every series as baselines.
type scraper struct {
	counters   map[string]float64
	histograms map[string]histogram
	// latest is replaced wholesale on every scrape, so a series the registry
	// stopped exposing disappears with it.
	latest latest
}

// newScraper returns a scraper with no baselines.
func newScraper() *scraper {
	return &scraper{
		counters:   make(map[string]float64),
		histograms: make(map[string]histogram),
		latest:     newLatest(),
	}
}

// scrape reads families into an aggregate and refreshes the absolute readings.
// Counters and histograms contribute how far they advanced since the previous
// scrape; a series seen for the first time only records its baseline. Gauges
// contribute their current reading.
func (s *scraper) scrape(families []*dto.MetricFamily) aggregate {
	observed := newAggregate()
	current := newLatest()

	for _, family := range families {
		name := family.GetName()

		switch name {
		case familySinkMessages:
			for _, m := range family.GetMetric() {
				status := domain.DeliveryStatus(labelValue(m, labelStatus))
				key := sinkKeyOf(m)

				value := m.GetCounter().GetValue()
				if current.sinks[key] == nil {
					current.sinks[key] = make(map[domain.DeliveryStatus]float64, 3)
				}
				current.sinks[key][status] += value

				if status != domain.DeliveryStatusSuccess && status != domain.DeliveryStatusFailure {
					continue
				}

				delta, ok := s.counterDelta(seriesKey(name, m), value)
				if !ok {
					continue
				}

				observed.sinkDelivery[key.sinkType] += delta
				if status == domain.DeliveryStatusSuccess {
					observed.delivered += delta
				} else {
					observed.failed += delta
				}
			}

		case familySinkDuration:
			for _, m := range family.GetMetric() {
				delta, ok := s.histogramDelta(seriesKey(name, m), histogramOf(m.GetHistogram()))
				if ok {
					observed.addLatency(sinkKeyOf(m), delta)
				}
			}

		case familyProcessedRows:
			for _, m := range family.GetMetric() {
				key := domain.TableIdentifier{
					Schema: domain.TableSchema(labelValue(m, labelSchema)),
					Name:   domain.TableName(labelValue(m, labelTable)),
				}
				ops := current.tables[key]
				addOperation(&ops, labelValue(m, labelOperation), float64(m.GetHistogram().GetSampleCount()))
				current.tables[key] = ops
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

	s.latest = current

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

// addOperation adds count to the field of ops that operation names, ignoring an
// operation the domain does not define.
func addOperation(ops *port.TableOps, operation string, count float64) {
	switch domain.OperationType(operation) {
	case domain.OperationTypeInsert:
		ops.Insert += count
	case domain.OperationTypeUpdate:
		ops.Update += count
	case domain.OperationTypeDelete:
		ops.Delete += count
	case domain.OperationTypeTruncate:
		ops.Truncate += count
	}
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

// sinkKeyOf reads the sink destination m belongs to off its labels.
func sinkKeyOf(m *dto.Metric) sinkKey {
	return sinkKey{
		sinkType: labelValue(m, labelType),
		host:     labelValue(m, labelHost),
		target:   labelValue(m, labelTarget),
	}
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

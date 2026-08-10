// Package metrics defines the Prometheus instruments emitted by the pipeline
// and the HTTP handler used to expose them on the telemetry server.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"hump/internal/domain"
)

// Metrics is the set of Prometheus instruments reported by the pipeline:
// replication lag gauges, per-row processing histograms, and per-sink
// delivery counters and latency histograms.
type Metrics struct {
	lagSenderSeconds prometheus.Gauge
	lagCommitSeconds prometheus.Gauge

	processedRowsDuration *prometheus.HistogramVec

	sinkMessages *prometheus.CounterVec
	sinkDuration *prometheus.HistogramVec

	limiterPending *prometheus.GaugeVec
}

// New builds a [Metrics] backed by a dedicated registry, prefixed with
// namespace, and returns it together with the [http.Handler] that exposes
// the registry over HTTP. The registry also collects standard Go runtime
// and process metrics.
func New(namespace string) (*Metrics, http.Handler) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsMemory, collectors.MetricsGC, collectors.MetricsScheduler),
		),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	factory := promauto.With(registry)

	metrics := &Metrics{
		lagSenderSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "lag_sender_seconds",
			Help:      "Transport lag between the source database server clock at the moment the message was sent and the moment the pipeline received it, in seconds. Reflects network and server-side decoding delay.",
		}),
		lagCommitSeconds: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "lag_commit_seconds",
			Help:      "End-to-end replication lag between the source transaction commit time and the moment the pipeline received the change, in seconds. Higher values indicate the consumer is falling behind the source.",
		}),

		processedRowsDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "processed_rows_duration",
			Help:      "Time spent processing a single row (change event) through the pipeline, in seconds, partitioned by operation, source database, schema, and table.",
		}, []string{"operation", "db", "schema", "table"}),

		sinkMessages: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sink_messages_total",
			Help:      "Total number of messages delivered to a sink, labeled by outcome (success or failure), sink type (e.g. kafka, http), connection host, and rendered target (topic/subject/exchange/stream/HTTP method).",
		}, []string{"status", "type", "host", "target"}),
		sinkDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "sink_duration_seconds",
			Help:      "Time spent dispatching a single message to a sink, in seconds, labeled by sink type, connection host, and rendered target (topic/subject/exchange/stream/HTTP method). Measures end-to-end write latency including any retries within a single attempt.",
		}, []string{"type", "host", "target"}),

		limiterPending: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "async_queue_total",
			Help:      "Number of items currently in the async queue per key, counting both items blocked waiting for a slot and items actively holding one. A persistently rising value means the sink for this key drains slower than events arrive and the backlog is consuming memory. WARNING: the key label is the rendered delivery.concurrencyKey — if that template includes per-row values, this metric has unbounded cardinality.",
		}, []string{"key"}),
	}

	return metrics, promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// LagSender records the transport lag between the source server clock at
// send time and the moment the pipeline received the message.
func (m *Metrics) LagSender(seconds float64) {
	m.lagSenderSeconds.Set(seconds)
}

// LagCommit records the end-to-end replication lag between the source
// transaction commit time and the moment the change was received.
func (m *Metrics) LagCommit(seconds float64) {
	m.lagCommitSeconds.Set(seconds)
}

// ProcessedRowsInsertDuration observes the processing duration of a single
// INSERT row for the given (db, schema, table).
func (m *Metrics) ProcessedRowsInsertDuration(value float64, db domain.DbName, schema domain.TableSchema, table domain.TableName) {
	m.processedRowsDuration.WithLabelValues("insert", string(db), string(schema), string(table)).Observe(value)
}

// ProcessedRowsUpdateDuration observes the processing duration of a single
// UPDATE row for the given (db, schema, table).
func (m *Metrics) ProcessedRowsUpdateDuration(value float64, db domain.DbName, schema domain.TableSchema, table domain.TableName) {
	m.processedRowsDuration.WithLabelValues("update", string(db), string(schema), string(table)).Observe(value)
}

// ProcessedRowsDeleteDuration observes the processing duration of a single
// DELETE row for the given (db, schema, table).
func (m *Metrics) ProcessedRowsDeleteDuration(value float64, db domain.DbName, schema domain.TableSchema, table domain.TableName) {
	m.processedRowsDuration.WithLabelValues("delete", string(db), string(schema), string(table)).Observe(value)
}

// ProcessedRowsTruncateDuration observes the processing duration of a
// TRUNCATE operation for the given (db, schema, table).
func (m *Metrics) ProcessedRowsTruncateDuration(value float64, db domain.DbName, schema domain.TableSchema, table domain.TableName) {
	m.processedRowsDuration.WithLabelValues("truncate", string(db), string(schema), string(table)).Observe(value)
}

// SinkMessagesSuccess increments the count of messages delivered
// successfully to the sink identified by sinkType, host, and target.
func (m *Metrics) SinkMessagesSuccess(value float64, sinkType, host, target string) {
	m.sinkMessages.WithLabelValues("success", sinkType, host, target).Add(value)
}

// SinkMessagesRetry increments the count of retried delivery attempts for
// the sink identified by sinkType, host, and target.
func (m *Metrics) SinkMessagesRetry(value float64, sinkType, host, target string) {
	m.sinkMessages.WithLabelValues("retry", sinkType, host, target).Add(value)
}

// SinkMessagesFailure increments the count of permanently failed deliveries
// for the sink identified by sinkType, host, and target.
func (m *Metrics) SinkMessagesFailure(value float64, sinkType, host, target string) {
	m.sinkMessages.WithLabelValues("failure", sinkType, host, target).Add(value)
}

// SinkDuration observes the time spent dispatching a single message to the
// sink identified by sinkType, host, and target.
func (m *Metrics) SinkDuration(duration float64, sinkType, host, target string) {
	m.sinkDuration.WithLabelValues(sinkType, host, target).Observe(duration)
}

// LimiterPendingInc records that one item entered the limiter for key
// (about to wait for or hold a concurrency slot).
func (m *Metrics) LimiterPendingInc(key string) {
	m.limiterPending.WithLabelValues(key).Inc()
}

// LimiterPendingDec records that one item left the limiter for key
// (released its slot, or abandoned the wait because ctx was done).
func (m *Metrics) LimiterPendingDec(key string) {
	m.limiterPending.WithLabelValues(key).Dec()
}

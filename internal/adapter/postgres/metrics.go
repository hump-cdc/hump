package postgres

import (
	"hump/internal/domain"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	replicationLag prometheus.Gauge
	rowDurations   *prometheus.HistogramVec
	rowObservers   map[rowObserverKey]prometheus.Observer
}

// rowObserverKey identifies the cached histogram observer for one operation on
// one table.
type rowObserverKey struct {
	operation domain.OperationType
	table     domain.TableIdentifier
}

func newMetrics(dbName string, factory promauto.Factory) *metrics {
	return &metrics{
		replicationLag: factory.NewGauge(prometheus.GaugeOpts{
			Name:        "replication_lag_bytes",
			Help:        "Replication lag in bytes between the current WAL write position on the source database and the WAL position the pipeline has already processed. Higher values indicate the pipeline is falling behind and the source is retaining WAL for the replication slot.",
			ConstLabels: prometheus.Labels{"db": dbName},
		}),
		rowDurations: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name: "processed_rows_duration_seconds",
			Help: "Time spent processing a single row (change event) through the pipeline, in seconds, partitioned by operation, source database, schema, and table.",
			Buckets: []float64{
				100e-6, // 0.1ms
				500e-6, // 0.5ms
				1e-3,   // 1ms
				10e-3,  // 10ms
				100e-3, // 100ms
				250e-3, // 250ms
				500e-3, // 500ms
				1,      // 1s
				3,      // 3s
				5,      // 5s
				10,     // 10s
			},
			ConstLabels: prometheus.Labels{"db": dbName},
		}, []string{"operation", "schema", "table"}),
		rowObservers: make(map[rowObserverKey]prometheus.Observer),
	}
}

// updateReplicationLag updates the replication lag metric with the difference between
// the last processed WAL position and the server's reported WAL end position.
func (m *metrics) updateReplicationLag(processed pglogrepl.LSN, server pglogrepl.LSN) {
	// LSN is unsigned, so the positions are compared before subtracting: a
	// processed position at or past the server's means no lag, not a wrapped one.
	if server > processed {
		m.replicationLag.Set(float64(server - processed))
	} else {
		m.replicationLag.Set(0)
	}
}

// observeRow records the duration of operation on table, caching the observer
// for the label set on first use.
func (m *metrics) observeRow(operation domain.OperationType, table domain.TableIdentifier, duration time.Duration) {
	key := rowObserverKey{operation: operation, table: table}

	observer, ok := m.rowObservers[key]
	if !ok {
		observer = m.rowDurations.WithLabelValues(string(operation), string(table.Schema), string(table.Name))
		m.rowObservers[key] = observer
	}

	observer.Observe(duration.Seconds())
}

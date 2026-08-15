package postgres

import (
	"hump/internal/domain"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	replicationLag       prometheus.Gauge
	rowDurations         *prometheus.HistogramVec
	rowInsertDurations   map[domain.TableIdentifier]prometheus.Observer
	rowUpdateDurations   map[domain.TableIdentifier]prometheus.Observer
	rowDeleteDurations   map[domain.TableIdentifier]prometheus.Observer
	rowTruncateDurations map[domain.TableIdentifier]prometheus.Observer
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
		rowInsertDurations:   make(map[domain.TableIdentifier]prometheus.Observer),
		rowUpdateDurations:   make(map[domain.TableIdentifier]prometheus.Observer),
		rowDeleteDurations:   make(map[domain.TableIdentifier]prometheus.Observer),
		rowTruncateDurations: make(map[domain.TableIdentifier]prometheus.Observer),
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

// observeInsert records the duration of an insert operation for a specific table.
func (m *metrics) observeInsert(table domain.TableIdentifier, duration time.Duration) {
	observer, ok := m.rowInsertDurations[table]
	if !ok {
		observer = m.rowDurations.WithLabelValues("insert", string(table.Schema), string(table.Name))
		m.rowInsertDurations[table] = observer
	}

	observer.Observe(duration.Seconds())
}

// observeInsert records the duration of an update operation for a specific table.
func (m *metrics) observeUpdate(table domain.TableIdentifier, duration time.Duration) {
	observer, ok := m.rowUpdateDurations[table]
	if !ok {
		observer = m.rowDurations.WithLabelValues("update", string(table.Schema), string(table.Name))
		m.rowUpdateDurations[table] = observer
	}

	observer.Observe(duration.Seconds())
}

// observeInsert records the duration of an delete operation for a specific table.
func (m *metrics) observeDelete(table domain.TableIdentifier, duration time.Duration) {
	observer, ok := m.rowDeleteDurations[table]
	if !ok {
		observer = m.rowDurations.WithLabelValues("delete", string(table.Schema), string(table.Name))
		m.rowDeleteDurations[table] = observer
	}

	observer.Observe(duration.Seconds())
}

// observeInsert records the duration of an truncate operation for a specific table.
func (m *metrics) observeTruncate(table domain.TableIdentifier, duration time.Duration) {
	observer, ok := m.rowTruncateDurations[table]
	if !ok {
		observer = m.rowDurations.WithLabelValues("truncate", string(table.Schema), string(table.Name))
		m.rowTruncateDurations[table] = observer
	}

	observer.Observe(duration.Seconds())
}

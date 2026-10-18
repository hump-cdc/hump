package snapshot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metrics holds the runner's Prometheus metrics.
type metrics struct {
	running prometheus.Gauge // running is 1 while a run is in progress, 0 otherwise.
}

// newMetrics registers the runner's metrics for dbName on factory.
func newMetrics(dbName string, factory promauto.Factory) *metrics {
	return &metrics{
		running: factory.NewGauge(prometheus.GaugeOpts{
			Name:        "snapshot_running",
			Help:        "Whether a snapshot run is in progress: 1 from its start, estimation included, until it finishes or is stopped; 0 otherwise.",
			ConstLabels: prometheus.Labels{"db": dbName},
		}),
	}
}

// setRunning sets the running gauge to 1 when running, 0 otherwise.
func (m *metrics) setRunning(running bool) {
	if running {
		m.running.Set(1)
		return
	}

	m.running.Set(0)
}

// Package metric builds the Prometheus registry the process exposes its
// metrics through.
package metric

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// New returns a [promauto.Factory] over a namespaced registry, the [http.Handler]
// exposing it and the registry itself as a [prometheus.Gatherer].
func New(namespace string) (promauto.Factory, http.Handler, prometheus.Gatherer) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsMemory, collectors.MetricsGC, collectors.MetricsScheduler),
		),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return promauto.With(prometheus.WrapRegistererWithPrefix(namespace+"_", registry)),
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		registry
}

package controller

import (
	"context"
	"errors"
	"fmt"
	"hump/pkg/app/shutdown"
	"hump/pkg/observability/log"
	"net/http"
)

// ObservabilityController is the HTTP-facing surface that serves the process
// health probes (/health/startup, /health/ready, /health/live) and the
// Prometheus /metrics scrape endpoint on a single port.
type ObservabilityController struct {
	healthHandler  http.Handler
	metricsHandler http.Handler
}

// NewObservabilityController constructs an [ObservabilityController] that
// serves the health handler under /health/startup, /health/ready and
// /health/live, and the metrics handler under /metrics.
func NewObservabilityController(healthHandler, metricsHandler http.Handler) *ObservabilityController {
	return &ObservabilityController{
		healthHandler:  healthHandler,
		metricsHandler: metricsHandler,
	}
}

// Serve binds to the given TCP port and blocks until the registered shutdown
// hook closes the underlying server. It returns nil on a clean shutdown and
// the server error otherwise.
func (c *ObservabilityController) Serve(port uint) error {
	logger := log.Scope("observability")

	mux := http.NewServeMux()

	mux.Handle("/health/startup", c.healthHandler)
	mux.Handle("/health/ready", c.healthHandler)
	mux.Handle("/health/live", c.healthHandler)
	mux.Handle("/metrics", c.metricsHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	shutdown.OnShutdown(shutdown.StageServers, "observability", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	logger.Info(fmt.Sprintf("Observability started at http://localhost:%d", port))
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

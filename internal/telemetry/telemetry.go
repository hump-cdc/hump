// Package telemetry exposes the HTTP server that publishes the process
// health probes (/health/startup, /health/ready, /health/live) and the
// Prometheus /metrics endpoint.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"hump/internal/app/shutdown"
	"hump/internal/log"
	"net/http"
)

var logger = log.Scope("telemetry")

// Telemetry is the HTTP-facing surface that serves health probes and the
// Prometheus scrape endpoint on a single port.
type Telemetry interface {
	// Serve binds to the given TCP port and blocks until the registered
	// shutdown hook closes the underlying server. It returns nil on a clean
	// shutdown and the server error otherwise.
	Serve(port uint) error
}

type telemetry struct {
	startupHandler http.Handler
	readyHandler   http.Handler
	liveHandler    http.Handler
	metricsHandler http.Handler
}

// New constructs a [Telemetry] that multiplexes the supplied handlers under
// /health/startup, /health/ready, /health/live, and /metrics respectively.
func New(startupHandler, readyHandler, liveHandler, metricsHandler http.Handler) Telemetry {
	return &telemetry{
		startupHandler: startupHandler,
		readyHandler:   readyHandler,
		liveHandler:    liveHandler,
		metricsHandler: metricsHandler,
	}
}

// Serve starts the telemetry HTTP server on the given port and registers a
// shutdown hook that gracefully stops it when the application shuts down.
func (t *telemetry) Serve(port uint) error {
	mux := http.NewServeMux()

	mux.Handle("/health/startup", t.startupHandler)
	mux.Handle("/health/ready", t.readyHandler)
	mux.Handle("/health/live", t.liveHandler)
	mux.Handle("/metrics", t.metricsHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	shutdown.OnShutdown("telemetry", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	logger.Debug(fmt.Sprintf("Telemetry started at http://localhost:%d", port))
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

package main

import (
	"context"
	"errors"
	"os"

	"hump/internal/app"
	"hump/internal/app/options"
	"hump/internal/log"
	"hump/internal/pipeline"
	"hump/internal/telemetry"
	"hump/internal/telemetry/health"
	"hump/internal/telemetry/metrics"
)

var logger = log.Scope("")

func main() {
	cfg, e := options.Parse()
	if errors.Is(e, options.ErrHelp) {
		os.Exit(0)
	}

	if e != nil {
		logger.Error(e.Error())
		os.Exit(1)
	}

	r, release := app.New(context.Background())
	defer release()

	health, healthStartupHandler, healthReadyHandler, healthLiveHandler := health.New()
	metrics, metricsHandler := metrics.New("hump")

	if cfg.TelemetryPort != nil {
		r.Run(
			// Telemetry
			func(ctx context.Context) error {
				t := telemetry.New(healthStartupHandler, healthReadyHandler, healthLiveHandler, metricsHandler)
				return t.Serve(*cfg.TelemetryPort)
			})
	}

	r.Run(
		// WAL
		func(ctx context.Context) error {
			p := pipeline.New(health, metrics)
			return p.Start(ctx, cfg.DatabaseUrl, cfg.Config, cfg.PublicationName, cfg.ReplicationSlotName)
		},
	)

	err := r.Wait()
	if err != nil {
		logger.Error(err.Error(), "error", err)
		os.Exit(1)
	} else {
		logger.Info("Good bye!")
	}
}

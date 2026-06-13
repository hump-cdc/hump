package main

import (
	"context"
	"errors"
	"os"

	"hump/internal/app"
	"hump/internal/app/options"
	"hump/internal/log"
	"hump/internal/pipeline"
	"hump/internal/source/postgres"
	"hump/internal/source/postgres/connection"
	"hump/internal/telemetry"
	"hump/internal/telemetry/health"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/errs"
	"hump/internal/web"
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

	if !cfg.Cleanup {
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

		if cfg.WebPort != nil {
			r.Run(
				// Web UI
				func(ctx context.Context) error {
					w := web.New(metricsHandler)
					return w.Serve(*cfg.WebPort)
				})
		}

		r.Run(
			// WAL
			func(ctx context.Context) error {
				p := pipeline.New(health, metrics)
				return p.Start(ctx, cfg.DatabaseUrl, cfg.Config, cfg.PublicationName, cfg.ReplicationSlotName)
			},
		)
	} else {
		r.Run(
			// Cleanup: drop publication + slot, then stop the runner.
			func(ctx context.Context) error {
				connector := connection.New(cfg.DatabaseUrl)
				if err := postgres.Cleanup(ctx, connector, cfg.PublicationName, cfg.ReplicationSlotName); err != nil {
					return err
				}

				logger.Info("Cleanup complete", "publication", cfg.PublicationName, "slot", cfg.ReplicationSlotName)
				return errs.ErrShutdown
			},
		)
	}

	err := r.Wait()
	if err != nil {
		logger.Error(err.Error(), "error", err)
		os.Exit(1)
	} else {
		logger.Info("Good bye!")
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cfg "hump/config"
	postgresAdapter "hump/internal/adapter/postgres"
	sinkAdapter "hump/internal/adapter/sink"
	processOperationFeature "hump/internal/application/process_operation"
	"hump/internal/controller"
	databasePort "hump/internal/port/database"
	sinkPort "hump/internal/port/sink"
	"hump/internal/telemetry"
	"hump/internal/telemetry/health"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/errs"
	"hump/internal/types/stream"
	"hump/internal/web"
	"hump/internal/web/streamer"
	"hump/pkg/app"
	"hump/pkg/app/options"
	"hump/pkg/telemetry/log"

	"github.com/goccy/go-yaml"
)

var logger = log.Scope("")

func main() {
	opts, e := options.Parse()
	if errors.Is(e, options.ErrHelp) {
		os.Exit(0)
	}

	if e != nil {
		logger.Error(e.Error())
		os.Exit(1)
	}

	r, release := app.New(context.Background())
	defer release()

	// Configuration
	configuration, err := cfg.Parse(opts.Config)
	if err != nil {
		logger.Error(err.Error(), "error", err)
		os.Exit(1)
		return
	}

	// Adapters
	var db databasePort.Database = postgresAdapter.NewDatabase(opts.DatabaseUrl, configuration, opts.PublicationName, opts.ReplicationSlotName)
	var sink sinkPort.Sink = sinkAdapter.New()

	if !opts.Cleanup {
		cfgResult, _ := yaml.Marshal(&configuration)
		logger.Debug(fmt.Sprintf("Configuration:\n%s\n", string(cfgResult)))

		for _, t := range configuration.Tables {
			logger.Debug("Configuration",
				"schema", t.Schema,
				"table", t.Name,
				"sinks", formatSinks(t.Sinks),
			)
		}

		// Services
		_, healthStartupHandler, healthReadyHandler, healthLiveHandler := health.New()
		metrics, metricsHandler := metrics.New("hump")

		// Shared operations stream fanned out to web SSE clients.
		operations := streamer.New[stream.Message]()

		// TODO: remove
		// r.Run(func(ctx context.Context) error {
		// 	ticker := time.NewTicker(5 * time.Second)
		// 	defer ticker.Stop()

		// 	reporter := newMetricsReporter(metrics)

		// 	for {
		// 		select {
		// 		case <-ctx.Done():
		// 			return nil
		// 		case <-ticker.C:
		// 			if err := reporter.report(); err != nil {
		// 				logger.Warn(err.Error(), "error", err)
		// 			}
		// 		}
		// 	}
		// })

		// Run
		if opts.TelemetryPort != nil {
			r.Run(
				// Telemetry
				func(ctx context.Context) error {
					t := telemetry.New(healthStartupHandler, healthReadyHandler, healthLiveHandler, metricsHandler)
					return t.Serve(*opts.TelemetryPort)
				})
		}

		if opts.WebPort != nil {
			r.Run(
				// Web UI
				func(ctx context.Context) error {
					w := web.New(opts, configuration, operations, metricsHandler)
					return w.Serve(ctx, *opts.WebPort)
				})
		}

		r.Run(
			// WAL
			func(ctx context.Context) error {
				process := processOperationFeature.NewUseCase(&configuration, sink, operations, metrics)
				controller := controller.NewProcessController(db, processOperationFeature.NewController(process))

				return controller.Listen(ctx)
			},
		)
	} else {
		r.Run(
			// Cleanup: drop publication + slot, then stop the runner.
			func(ctx context.Context) error {
				if err := db.Cleanup(ctx); err != nil {
					return err
				}

				return errs.ErrShutdown
			},
		)
	}

	err = r.Wait()
	if err != nil {
		logger.Error(err.Error(), "error", err)
		os.Exit(1)
	} else {
		logger.Info("Good bye!")
	}
}

func formatSinks(sinks []sinkPort.Config) string {
	parts := make([]string, 0, len(sinks))
	for _, s := range sinks {
		parts = append(parts, s.Sink.DisplayName())
	}
	return strings.Join(parts, ", ")
}

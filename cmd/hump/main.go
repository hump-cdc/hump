package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cfg "hump/config"
	sinkcfg "hump/config/sink"
	postgresAdapter "hump/internal/adapter/postgres"
	sinkAdapter "hump/internal/adapter/sink"
	processOperationFeature "hump/internal/application/process_operation"
	"hump/internal/controller"
	databasePort "hump/internal/port/database"
	sinkPort "hump/internal/port/sink"
	"hump/pkg/app"
	"hump/pkg/app/options"
	"hump/pkg/telemetry/health"
	"hump/pkg/telemetry/log"
	"hump/pkg/telemetry/metric"

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

	// Telemetry
	health, err := health.New("hump")
	if err != nil {
		logger.Error("Unable to init health probes", "error", err)
		os.Exit(1)
		return
	}
	metrics, metricHandler := metric.New("hump")

	// Adapters
	pg := postgresAdapter.New(opts.DatabaseUrl, configuration, opts.PublicationName, opts.ReplicationSlotName, health.Postgres, health.WalSender, metrics)
	var db databasePort.Database = pg
	var sink sinkPort.Sink = sinkAdapter.New()

	// Application
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

		// Run
		if opts.TelemetryPort != nil {
			r.Serve(
				// Telemetry
				func(ctx context.Context) error {
					t := controller.NewTelemetryController(health.Handler(), metricHandler)
					return t.Serve(*opts.TelemetryPort)
				})
		}

		// Web UI
		if opts.WebPort != nil {
			r.Serve(
				// Web UI
				func(ctx context.Context) error {
					// configController := getconfig.NewController(opts, configuration, db)
					server := controller.NewWebUIController()
					return server.Serve(ctx, *opts.WebPort)
				})
		}

		// WAL
		r.Run(
			func(ctx context.Context) error {
				processOperationUseCase := processOperationFeature.NewUseCase(&configuration, sink, metrics)
				controller := controller.NewProcessController(db, processOperationFeature.NewController(processOperationUseCase))

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

				return app.ErrShutdown
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

func formatSinks(sinks []sinkcfg.Config) string {
	parts := make([]string, 0, len(sinks))
	for _, s := range sinks {
		parts = append(parts, s.Sink.DisplayName())
	}
	return strings.Join(parts, ", ")
}

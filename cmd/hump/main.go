package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cfg "hump/config"
	sinkcfg "hump/config/sink"
	liveStreamerAdapter "hump/internal/adapter/live_streamer"
	metricsCollectorAdapter "hump/internal/adapter/metrics_collector"
	operationsCollectorAdapter "hump/internal/adapter/operations_collector"
	postgresAdapter "hump/internal/adapter/postgres"
	sinkAdapter "hump/internal/adapter/sink"
	snapshotAdapter "hump/internal/adapter/snapshot"
	processOperationFeature "hump/internal/application/processing/process_operation"
	getConfigFeature "hump/internal/application/webui/get_config"
	getMetricsFeature "hump/internal/application/webui/get_metrics"
	getOperationsFeature "hump/internal/application/webui/get_operations"
	getSnapshotEstimatesFeature "hump/internal/application/webui/get_snapshot_estimates"
	getSnapshotTablesFeature "hump/internal/application/webui/get_snapshot_tables"
	startSnapshotFeature "hump/internal/application/webui/start_snapshot"
	stopSnapshotFeature "hump/internal/application/webui/stop_snapshot"
	streamSnapshotFeature "hump/internal/application/webui/stream_snapshot"
	streamUpdatesFeature "hump/internal/application/webui/stream_updates"
	"hump/internal/controller"
	"hump/internal/domain"
	databasePort "hump/internal/port/database"
	livestreamPort "hump/internal/port/live_stream"
	metricsCollectorPort "hump/internal/port/metrics_collector"
	operationsCollectorPort "hump/internal/port/operations_collector"
	sinkPort "hump/internal/port/sink"
	snapshotPort "hump/internal/port/snapshot"
	"hump/pkg/app"
	"hump/pkg/app/options"
	"hump/pkg/observability/health"
	"hump/pkg/observability/log"
	"hump/pkg/observability/metric"

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

	// Observability
	health, err := health.New("hump")
	if err != nil {
		logger.Error("Unable to init health probes", "error", err)
		os.Exit(1)
		return
	}
	metrics, metricHandler, metricGatherer := metric.New("hump")

	// Adapters
	var db databasePort.Database = postgresAdapter.New(opts.DatabaseUrl, configuration, opts.PublicationName, opts.ReplicationSlotName, health.Postgres, health.WalSender, metrics)
	var sink sinkPort.Sink = sinkAdapter.New()
	var collector metricsCollectorPort.Collector
	if opts.WebPort != nil {
		collector = metricsCollectorAdapter.New(metricGatherer,
			metricsCollectorAdapter.WithTopology(topology(configuration)))
	}

	// The operations log only serves the web UI, and it stays idle, holding no
	// buffer at all, until a client connects. The live streamer wraps it, so the
	// pipeline keeps writing to one collector while the SSE stream reads the
	// same deliveries off the wrapper.
	var operations operationsCollectorPort.Collector
	var live livestreamPort.Streamer
	if opts.WebPort != nil {
		streamer := liveStreamerAdapter.New(
			operationsCollectorAdapter.New(),
			liveStreamerAdapter.WithInterval(500*time.Millisecond),
			liveStreamerAdapter.WithScraper(collector),
		)
		operations, live = streamer, streamer
	}

	// One processing use case serves both WAL and snapshot runs: the snapshot
	// pauses WAL delivery, so the two never call it at once.
	var processOperation *processOperationFeature.Controller
	var snapshot snapshotPort.Runner
	if !opts.Cleanup {
		processOperation = processOperationFeature.NewController(processOperationFeature.NewUseCase(&configuration, db.GetDbName(), sink, metrics, collector, operations))

		if opts.WebPort != nil {
			snapshot = snapshotAdapter.New(db, configuration, processOperation.Process, metrics)
		}
	}

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
				// Observability
				func(ctx context.Context) error {
					t := controller.NewObservabilityController(health.Handler(), metricHandler)
					return t.Serve(*opts.TelemetryPort)
				})
		}

		// Web UI
		if opts.WebPort != nil {
			r.Serve(
				// Web UI
				func(ctx context.Context) error {
					server := controller.NewWebUIController(
						getConfigFeature.NewController(getConfigFeature.NewUseCase(opts, configuration, db)),
						getMetricsFeature.NewController(getMetricsFeature.NewUseCase(collector)),
						getOperationsFeature.NewController(getOperationsFeature.NewUseCase(operations)),
						streamUpdatesFeature.NewController(streamUpdatesFeature.NewUseCase(collector, operations, live, snapshot)),
						getSnapshotEstimatesFeature.NewController(getSnapshotEstimatesFeature.NewUseCase(db)),
						getSnapshotTablesFeature.NewController(getSnapshotTablesFeature.NewUseCase(configuration, db)),
						streamSnapshotFeature.NewController(streamSnapshotFeature.NewUseCase(snapshot)),
						startSnapshotFeature.NewController(startSnapshotFeature.NewUseCase(snapshot)),
						stopSnapshotFeature.NewController(stopSnapshotFeature.NewUseCase(snapshot)),
					)
					return server.Serve(ctx, *opts.WebPort)
				})
		}

		// WAL
		r.Run(
			func(ctx context.Context) error {
				controller := controller.NewProcessController(db, processOperation)

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

// topology resolves every table-to-sink-target edge of configuration, rendering
// host and target the way the bindings do, so both match the metric labels.
func topology(configuration cfg.Configuration) []metricsCollectorPort.Connection {
	conns := make([]metricsCollectorPort.Connection, 0, len(configuration.Tables))

	for _, table := range configuration.ResolveAll() {
		for _, sc := range table.Sinks {
			if sc.Sink == nil {
				continue
			}

			host, target := sc.Sink.Metadata(sinkPort.SinkContext{Schema: table.Schema, Table: table.Name})
			conns = append(conns, metricsCollectorPort.Connection{
				Table:    domain.TableIdentifier{Schema: table.Schema, Name: table.Name},
				SinkType: string(sc.Sink.Type()),
				Host:     host,
				Target:   target,
			})
		}
	}

	return conns
}

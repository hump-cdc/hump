// Package process_operation delivers one WAL row change to every sink
// configured for its table, in parallel, returning once all of them finish.
package process_operation

import (
	"context"
	"fmt"
	"time"

	"hump/config"
	"hump/internal/domain"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"
	"hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/version"
	workerpool "hump/pkg/worker_pool"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Configuration resolves the configured tables with their sinks.
type Configuration interface {
	ResolveAll() []config.Table
}

// Observer counts the deliveries of one connection for the dashboard. It is the
// part of the metrics collector this package needs, and is nil when the web UI
// is disabled.
type Observer interface {
	Record(conn metricscollector.Connection, status domain.DeliveryStatus, n int)
}

// OperationLog records finished sink deliveries, retries and all, for the
// dashboard's operations view. It is nil when the web UI is disabled.
type OperationLog interface {
	// Connected reports whether anything is watching, so a delivery can skip
	// the work of describing itself when nothing is.
	Connected() bool
	Record(rec *operationscollector.Record)
}

// UseCase fans row changes out to the sink bindings of their table.
type UseCase struct {
	bindings map[domain.TableIdentifier][]*binding
	wp       *workerpool.WorkerPool[*sink.Context]
}

// NewUseCase builds the sink bindings of every configured table, labelling its
// sink metrics with the source database dbName.
func NewUseCase(cfg Configuration, dbName string, sinksPort sink.Sink, metricsFactory promauto.Factory, observer Observer, opLog OperationLog) *UseCase {
	workerCount := 0

	// Metrics
	sinkMessagesCounter := metricsFactory.NewCounterVec(prometheus.CounterOpts{
		Name:        "sink_messages_total",
		Help:        "Total number of messages delivered to a sink, labeled by source database, outcome (success or failure), sink type (e.g. kafka, http), connection host, and rendered target (topic/subject/exchange/stream/URL).",
		ConstLabels: prometheus.Labels{"db": dbName},
	}, []string{"status", "type", "host", "target"})

	sinkDurationHistogram := metricsFactory.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "sink_duration_seconds",
		Help:        "Time spent dispatching a single message to a sink, in seconds, labeled by source database, sink type, connection host, and rendered target (topic/subject/exchange/stream/URL). Measures end-to-end write latency including any retries within a single attempt.",
		ConstLabels: prometheus.Labels{"db": dbName},
	}, []string{"type", "host", "target"})

	// Build binding mappings per (schema, table) pair
	bindings := make(map[domain.TableIdentifier][]*binding)
	for _, table := range cfg.ResolveAll() {
		t := domain.TableIdentifier{Schema: table.Schema, Name: table.Name}

		tableBindings := make([]*binding, 0, len(table.Sinks))
		for _, sinkCfg := range table.Sinks {
			b := newBinding(sinkCfg, table.Schema, table.Name, sinksPort, sinkMessagesCounter, sinkDurationHistogram, observer, opLog)
			tableBindings = append(tableBindings, b)
		}

		bindings[t] = append(bindings[t], tableBindings...)

		if len(bindings[t]) > workerCount {
			workerCount = len(bindings[t])
		}
	}

	// Setup the worker pool with the number of workers equal to the maximum number of sinks,
	// at least one: the pool rejects zero, which no tables or a table without sinks would give
	workerCount = max(workerCount, 1)
	wp := workerpool.New[*sink.Context](
		workerpool.WithWorkers(workerCount),
		workerpool.WithQueueSize(workerCount),
		workerpool.WithErrorBufferSize(1024),
		workerpool.WithBackpressurePolicy(workerpool.BackpressureBlock),
	)

	go func() {
		// Print worker errors to the logger.
		// This goroutine will run until the worker pool is shut down.
		for err := range wp.Errors() {
			logger.Error("Worker error", "error", err)
		}
	}()

	shutdown.OnShutdown(shutdown.StageWorkers, "process_operation", func(_ context.Context) error {
		return wp.Shutdown()
	})

	return &UseCase{
		bindings: bindings,
		wp:       wp,
	}
}

// Process resolves the configured sinks for the (schema, table) pair and
// delivers the WAL operation to each of them in parallel.
func (uc *UseCase) Process(
	ctx context.Context,

	timestamp time.Time,
	operation domain.OperationType,

	dbName domain.DbName,
	schema domain.TableSchema,
	table domain.TableName,

	headers domain.Headers,
	prev domain.DbRecord,
	next domain.DbRecord,
) error {
	traceparent := headers["traceparent"]
	logger := logger.WithTraceparent(traceparent)

	logger.Debug("Processing",
		"operation", operation, "database", dbName, "schema", schema, "table", table,
	)

	operationContext := &sink.Context{
		Version: version.Version,

		Timestamp: timestamp,
		Operation: operation,

		Database: dbName,
		Schema:   schema,
		Table:    table,

		Headers: headers,

		Prev: prev,
		Next: next,
	}

	bindings := uc.bindings[domain.TableIdentifier{Schema: schema, Name: table}]
	var submitErr error

	if err := uc.wp.BeginBatch(); err != nil {
		return fmt.Errorf("begin batch: %w", err)
	}
	for _, binding := range bindings {
		if !binding.accepts(operation) {
			continue
		}

		if err := uc.wp.Submit(ctx, binding.fn, operationContext); err != nil {
			logger.Error("Failed to submit task to worker pool",
				"operation", operation, "database", dbName, "schema", schema, "table", table, "error", err,
			)
			submitErr = err
		}
	}
	err := uc.wp.FlushBatch()

	if submitErr != nil {
		return submitErr
	}

	if err == nil {
		logger.Debug("Processed",
			"operation", operation, "database", dbName, "schema", schema, "table", table,
		)
	} else {
		logger.Error("Failed to process",
			"operation", operation, "database", dbName, "schema", schema, "table", table,
			"error", err,
		)
	}

	return err
}

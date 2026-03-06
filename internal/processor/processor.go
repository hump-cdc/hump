package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hump/internal/log"
	asynclimiter "hump/internal/processor/async_limiter"
	"hump/internal/processor/retry"
	"hump/internal/sinks/http"
	"hump/internal/sinks/kafka"
	"hump/internal/sinks/nats"
	"hump/internal/sinks/rabbitmq"
	"hump/internal/sinks/rabbitmq_stream"
	"hump/internal/source/postgres/introspection"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/config"
	"hump/internal/types/errs"
	"hump/internal/types/sink"
	"hump/internal/types/wal"
	"hump/pkg/version"
)

// processor dispatches a single WAL change to every sink configured for the
// originating table, applying per-target retry and concurrency control.
type processor struct {
	introspector *introspection.Introspector
	metrics      *metrics.Metrics
	asyncLimiter *asynclimiter.AsyncLimiter
}

func New(introspector *introspection.Introspector, metrics *metrics.Metrics) *processor {
	return &processor{
		introspector: introspector,
		metrics:      metrics,
		asyncLimiter: asynclimiter.New(metrics),
	}
}

// Process resolves the configured sinks for the (schema, table) pair and
// delivers the WAL operation to each of them in parallel, returning only after
// every fan-out goroutine has finished.
func (p *processor) Process(
	ctx context.Context,

	timestamp time.Time,
	operation wal.WalOperation,

	dbName string,
	schema string,
	table string,

	headers map[string]string,
	prev map[string]any,
	next map[string]any,
) (string, string, string, error) {
	traceparent := headers["traceparent"]
	logger := log.Scope("processor").WithTraceparent(traceparent)

	logger.Debug("Processing",
		"operation", operation,
		"database", dbName,
		"schema", schema,
		"table", table,
		"operation", operation,
		"prev", prev,
		"next", next,
	)

	ref, targets, err := p.introspector.ResolveTableConfiguration(ctx, schema, table)
	if err != nil {
		return dbName, "", "", fmt.Errorf("failed to resolve table configuration for %s.%s: %w", schema, table, err)
	}

	resolvedSchema, resolvedTable := ref.Schema, ref.Name

	var wg sync.WaitGroup
	r := retry.New(&wg, traceparent, p.asyncLimiter, p.metrics)

	for _, target := range targets {
		operationContext := sink.SinkContext{
			Version: version.Version,

			SinkType: string(target.Sink.Type()),
			SinkName: target.Sink.DisplayName(),

			Timestamp: timestamp,
			Operation: operation,

			Database: dbName,
			Schema:   resolvedSchema,
			Table:    resolvedTable,

			Headers: headers,

			Prev: prev,
			Next: next,
		}

		concurrencyKey, _ := target.Delivery.ConcurrencyKey.Render(operationContext)
		if len(concurrencyKey) == 0 {
			concurrencyKey = fmt.Sprintf("%s(%s)", operationContext.SinkType, operationContext.SinkName)
		}

		wg.Add(1)
		go r.Process(
			ctx, target, concurrencyKey,
			func(ctx context.Context) error {
				var err error

				switch sink := target.Sink.(type) {
				case *config.KafkaSink:
					err = kafka.Process(ctx, *sink, &operationContext)
				case *config.NatsCoreSink:
					err = nats.ProcessCore(ctx, *sink, &operationContext)
				case *config.NatsJetStreamSink:
					err = nats.ProcessJetStream(ctx, *sink, &operationContext)
				case *config.HTTPSink:
					err = http.Process(ctx, *sink, &operationContext)
				case *config.RabbitMQSink:
					err = rabbitmq.Process(ctx, *sink, &operationContext)
				case *config.RabbitMQStreamSink:
					err = rabbitmq_stream.Process(ctx, *sink, &operationContext)
				default:
					return errs.ErrUnknownSinkType
				}

				return err
			})
	}
	wg.Wait()

	if r.Error() == nil {
		logger.Debug("Processed",
			"operation", operation,
			"database", dbName,
			"schema", schema,
			"table", table,
			"operation", operation,
			"prev", prev,
			"next", next,
		)
	} else {
		logger.Error("Failed to process",
			"operation", operation,
			"database", dbName,
			"schema", schema,
			"table", table,
			"operation", operation,
			"prev", prev,
			"next", next,
			"errror", r.Error(),
		)

	}

	return dbName, resolvedSchema, resolvedTable, r.Error()
}

package processor

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
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
	"hump/internal/types/stream"
	"hump/internal/types/wal"
	"hump/pkg/version"
)

// processor dispatches a single WAL change to every sink configured for the
// originating table, applying per-target retry and concurrency control.
type processor struct {
	introspector *introspection.Introspector
	publisher    stream.Publisher
	metrics      *metrics.Metrics
	asyncLimiter *asynclimiter.AsyncLimiter
	seq          atomic.Uint64
}

func New(introspector *introspection.Introspector, publisher stream.Publisher, metrics *metrics.Metrics) *processor {
	return &processor{
		introspector: introspector,
		publisher:    publisher,
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

		metaHost, metaTarget := target.Sink.Metadata(operationContext)

		// Only stream (and pay the per-sink render cost) when a web client is
		// actually subscribed to the operations stream. When streaming, the base
		// event carries a per-operation correlation id; the retry layer emits the
		// operation event and any retry events after each publish attempt.
		var pub stream.Publisher
		var streamEvent stream.Event
		if p.publisher != nil && p.publisher.HasSubscribers() {
			pub = p.publisher
			streamEvent = stream.Event{
				ID:        strconv.FormatUint(p.seq.Add(1), 10),
				Timestamp: timestamp,
				Schema:    resolvedSchema,
				Table:     resolvedTable,
				Operation: operation,
				SinkType:  string(target.Sink.Type()),
				Host:      metaHost,
				Target:    metaTarget,
			}
		}

		wg.Add(1)
		go r.Process(
			ctx, target, concurrencyKey, metaHost, metaTarget, streamEvent, pub,
			func(ctx context.Context, streamEvent *stream.Event) error {
				var err error

				switch sink := target.Sink.(type) {
				case *config.KafkaSink:
					err = kafka.Process(ctx, *sink, &operationContext, streamEvent)
				case *config.NatsCoreSink:
					err = nats.ProcessCore(ctx, *sink, &operationContext, streamEvent)
				case *config.NatsJetStreamSink:
					err = nats.ProcessJetStream(ctx, *sink, &operationContext, streamEvent)
				case *config.HTTPSink:
					err = http.Process(ctx, *sink, &operationContext, streamEvent)
				case *config.RabbitMQSink:
					err = rabbitmq.Process(ctx, *sink, &operationContext, streamEvent)
				case *config.RabbitMQStreamSink:
					err = rabbitmq_stream.Process(ctx, *sink, &operationContext, streamEvent)
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

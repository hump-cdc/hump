package process_operation

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"hump/internal/domain"
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
	"hump/pkg/limit"
	"hump/pkg/retry"
	"hump/pkg/telemetry/log"
	"hump/pkg/version"
)

// UseCase dispatches a single WAL change to every sink configured for the
// originating table, applying per-target retry and concurrency control.
type UseCase struct {
	introspector *introspection.Introspector
	publisher    stream.Publisher
	metrics      *metrics.Metrics
	limiter      *limit.Limit
	seq          atomic.Uint64
}

func NewUseCase(introspector *introspection.Introspector, publisher stream.Publisher, metrics *metrics.Metrics) *UseCase {
	return &UseCase{
		introspector: introspector,
		publisher:    publisher,
		metrics:      metrics,
		limiter:      limit.New(metrics),
	}
}

// Process resolves the configured sinks for the (schema, table) pair and
// delivers the WAL operation to each of them in parallel, returning only after
// every fan-out goroutine has finished.
func (p *UseCase) Process(
	ctx context.Context,

	timestamp time.Time,
	operation domain.OperationType,

	dbName domain.DbName,
	schema domain.TableSchema,
	table domain.TableName,

	headers domain.Headers,
	prev domain.DbRecord,
	next domain.DbRecord,
) (domain.DbName, domain.TableSchema, domain.TableName, error) {
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
	r := retry.New(&wg, traceparent, p.limiter, p.metrics)

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

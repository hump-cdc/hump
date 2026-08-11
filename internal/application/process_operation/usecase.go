package process_operation

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"hump/internal/domain"
	"hump/internal/port/sink"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/stream"
	"hump/pkg/limit"
	"hump/pkg/retry"
	"hump/pkg/telemetry/log"
	"hump/pkg/version"
)

type SinkConfigurationResolver interface {
	Resolve(schema domain.TableSchema, table domain.TableName) []sink.Config
}

type UseCase struct {
	sinkConfigurationResolver SinkConfigurationResolver
	sinks                     sink.Sink
	publisher                 stream.Publisher
	metrics                   *metrics.Metrics
	limiter                   *limit.Limit
	seq                       atomic.Uint64
}

func NewUseCase(sinkConfigurationResolver SinkConfigurationResolver, sinks sink.Sink, publisher stream.Publisher, metrics *metrics.Metrics) *UseCase {
	return &UseCase{
		sinkConfigurationResolver: sinkConfigurationResolver,
		sinks:                     sinks,
		publisher:                 publisher,
		metrics:                   metrics,
		limiter:                   limit.New(metrics),
	}
}

// Process resolves the configured sinks for the (schema, table) pair and
// delivers the WAL operation to each of them in parallel, returning only after
// every fan-out goroutine has finished.
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

	targets := uc.sinkConfigurationResolver.Resolve(schema, table)
	if targets == nil {
		return nil
	}

	var wg sync.WaitGroup
	r := retry.New(&wg, traceparent, uc.limiter, uc.metrics)

	for _, target := range targets {
		operationContext := sink.SinkContext{
			Version: version.Version,

			SinkType: string(target.Sink.Type()),
			SinkName: target.Sink.DisplayName(),

			Timestamp: timestamp,
			Operation: operation,

			Database: dbName,
			Schema:   schema,
			Table:    table,

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
		if uc.publisher != nil && uc.publisher.HasSubscribers() {
			pub = uc.publisher
			streamEvent = stream.Event{
				ID:        strconv.FormatUint(uc.seq.Add(1), 10),
				Timestamp: timestamp,
				Schema:    schema,
				Table:     table,
				Operation: operation,
				SinkType:  string(target.Sink.Type()),
				Host:      metaHost,
				Target:    metaTarget,
			}
		}

		wg.Add(1)
		go r.Process(
			ctx,
			target.Failure, target.Retry, target.Delivery,
			operationContext.SinkType, operationContext.SinkName,
			concurrencyKey, metaHost, metaTarget, streamEvent, pub,
			func(ctx context.Context, streamEvent *stream.Event) error {
				return uc.sinks.Process(ctx, target.Sink, &operationContext, streamEvent)
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

	return r.Error()
}

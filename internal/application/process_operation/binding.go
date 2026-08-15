package process_operation

import (
	"context"
	"fmt"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/sink"
	"hump/pkg/resilience"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type binding struct {
	sinkName              string
	table                 string
	fn                    func(ctx context.Context, operationContext *sink.SinkContext) error
	messageCounterSuccess prometheus.Counter
	messageCounterRetry   prometheus.Counter
	messageCounterFailure prometheus.Counter
	durationHistogram     prometheus.Observer
}

func newBinding(
	cfg sinkcfg.Config,
	schema domain.TableSchema,
	tableName domain.TableName,
	sinksPort sink.Sink,
	sinkMessagesCounter *prometheus.CounterVec,
	sinkDurationHistogram *prometheus.HistogramVec,
) *binding {
	host, target := cfg.Sink.Metadata(sink.SinkContext{})
	b := &binding{
		sinkName:              string(cfg.Sink.Type()),
		table:                 fmt.Sprintf("%s.%s", schema, tableName),
		messageCounterSuccess: sinkMessagesCounter.WithLabelValues("success", string(cfg.Sink.Type()), host, target),
		messageCounterRetry:   sinkMessagesCounter.WithLabelValues("retry", string(cfg.Sink.Type()), host, target),
		messageCounterFailure: sinkMessagesCounter.WithLabelValues("failure", string(cfg.Sink.Type()), host, target),
		durationHistogram:     sinkDurationHistogram.WithLabelValues(string(cfg.Sink.Type()), host, target),
	}

	fn := func(ctx context.Context, operationContext *sink.SinkContext) error {
		return sinksPort.Process(ctx, cfg.Sink, operationContext)
	}
	fn = resilience.WithTimeout(fn, cfg.Retry.AttemptTimeout)
	fn = resilience.WithRetry(fn, uint(cfg.Retry.MaxAttempts), cfg.Retry.Backoff.Policy(), b)
	fn = resilience.WithStatus(fn, b)
	b.fn = fn

	return b
}

func (b *binding) Process(ctx context.Context, operationContext *sink.SinkContext) error {
	return b.fn(ctx, operationContext)
}

// OnDuration reports how long one delivery attempt took.
func (b *binding) OnDuration(elapsed time.Duration) {
	b.durationHistogram.Observe(elapsed.Seconds())
}

// OnSuccess reports a delivery that finished without an error.
func (b *binding) OnSuccess() {
	b.messageCounterSuccess.Add(1)
}

// OnRetry reports that a failed attempt is about to be repeated.
func (b *binding) OnRetry() {
	b.messageCounterRetry.Add(1)
}

// OnFailure reports a delivery that exhausted its attempts.
func (b *binding) OnFailure() {
	b.messageCounterFailure.Add(1)
}

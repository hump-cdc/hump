// Package sink dispatches one rendered row change to the broker-specific sink
// package its configured definition selects.
package sink

import (
	"context"

	sinkcfg "hump/config/sink"
	"hump/internal/adapter/sink/http"
	"hump/internal/adapter/sink/kafka"
	"hump/internal/adapter/sink/nats"
	"hump/internal/adapter/sink/rabbitmq"
	"hump/internal/adapter/sink/rabbitmq_stream"
	p "hump/internal/port/sink"
)

// Sink implements the sink port by routing each delivery to the sink package
// matching its definition type.
type Sink struct {
}

// New returns a Sink.
func New() *Sink {
	return &Sink{}
}

// Process delivers operationContext to the sink cfg describes, recording the
// rendered message on capture when it is non-nil.
func (s *Sink) Process(ctx context.Context, cfg p.Definition, operationContext *p.Context, capture *p.Capture) error {
	var err error

	switch sink := cfg.(type) {
	case *sinkcfg.KafkaSink:
		err = kafka.Process(ctx, *sink, operationContext, capture)
	case *sinkcfg.NatsCoreSink:
		err = nats.ProcessCore(ctx, *sink, operationContext, capture)
	case *sinkcfg.NatsJetStreamSink:
		err = nats.ProcessJetStream(ctx, *sink, operationContext, capture)
	case *sinkcfg.HTTPSink:
		err = http.Process(ctx, *sink, operationContext, capture)
	case *sinkcfg.RabbitMQSink:
		err = rabbitmq.Process(ctx, *sink, operationContext, capture)
	case *sinkcfg.RabbitMQStreamSink:
		err = rabbitmq_stream.Process(ctx, *sink, operationContext, capture)
	default:
		return p.ErrUnknownSinkType
	}

	return err
}

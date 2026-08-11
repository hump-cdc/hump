package sink

import (
	"context"

	"hump/internal/adapter/sink/http"
	"hump/internal/adapter/sink/kafka"
	"hump/internal/adapter/sink/nats"
	"hump/internal/adapter/sink/rabbitmq"
	"hump/internal/adapter/sink/rabbitmq_stream"
	p "hump/internal/port/sink"
	"hump/internal/types/stream"
)

type Sink struct{}

func New() *Sink {
	return &Sink{}
}

func (s *Sink) Process(ctx context.Context, cfg p.Definition, operationContext *p.SinkContext, streamEvent *stream.Event) error {
	var err error

	switch sink := cfg.(type) {
	case *p.KafkaSink:
		err = kafka.Process(ctx, *sink, operationContext, streamEvent)
	case *p.NatsCoreSink:
		err = nats.ProcessCore(ctx, *sink, operationContext, streamEvent)
	case *p.NatsJetStreamSink:
		err = nats.ProcessJetStream(ctx, *sink, operationContext, streamEvent)
	case *p.HTTPSink:
		err = http.Process(ctx, *sink, operationContext, streamEvent)
	case *p.RabbitMQSink:
		err = rabbitmq.Process(ctx, *sink, operationContext, streamEvent)
	case *p.RabbitMQStreamSink:
		err = rabbitmq_stream.Process(ctx, *sink, operationContext, streamEvent)
	default:
		return p.ErrUnknownSinkType
	}

	return err
}

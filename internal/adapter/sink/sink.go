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

type Sink struct {
}

func New() *Sink {
	return &Sink{}
}

func (s *Sink) Process(ctx context.Context, cfg p.Definition, operationContext *p.SinkContext) error {
	var err error

	switch sink := cfg.(type) {
	case *sinkcfg.KafkaSink:
		err = kafka.Process(ctx, *sink, operationContext)
	case *sinkcfg.NatsCoreSink:
		err = nats.ProcessCore(ctx, *sink, operationContext)
	case *sinkcfg.NatsJetStreamSink:
		err = nats.ProcessJetStream(ctx, *sink, operationContext)
	case *sinkcfg.HTTPSink:
		err = http.Process(ctx, *sink, operationContext)
	case *sinkcfg.RabbitMQSink:
		err = rabbitmq.Process(ctx, *sink, operationContext)
	case *sinkcfg.RabbitMQStreamSink:
		err = rabbitmq_stream.Process(ctx, *sink, operationContext)
	default:
		return p.ErrUnknownSinkType
	}

	return err
}

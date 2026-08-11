// Package rabbitmq_stream implements the RabbitMQ Streams sink for hump.
package rabbitmq_stream

import (
	"context"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"

	sink "hump/internal/port/sink"
	"hump/internal/types/stream"
)

// Process delivers a single WAL operation to a RabbitMQ Streams sink. It
// renders the message, ensures the producer for the rendered stream exists
// (running the per-(URL, stream) reconciler on first use), then publishes
// under the producer's serialization mutex and waits for the publisher
// confirmation. When streamEvent is non-nil the rendered message's fields are recorded
// on it for the operations stream.
func Process(ctx context.Context, cfg sink.RabbitMQStreamSink, operationContext *sink.SinkContext, streamEvent *stream.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	streamName, payload, headers, err := renderAll(cfg, operationContext)
	if err != nil {
		return wrapError(err)
	}

	if streamEvent != nil {
		streamEvent.Payload = payload
		streamEvent.Headers = headers
	}

	rp, err := getProducer(ctx, cfg, operationContext, streamName)
	if err != nil {
		return wrapError(err)
	}

	msg := amqp.NewMessage([]byte(payload))
	msg.ApplicationProperties = toAMQPProps(headers)

	// Serialize per producer so the NotifyPublishConfirmation channel set up
	// below isn't replaced out from under us by a concurrent publish — see
	// rabbitProducer.publishMu.
	rp.publishMu.Lock()
	defer rp.publishMu.Unlock()

	confirmCh := rp.prod.NotifyPublishConfirmation()

	if err := rp.prod.Send(msg); err != nil {
		return wrapError(err)
	}

	select {
	case confs := <-confirmCh:
		if len(confs) == 0 || !confs[0].IsConfirmed() {
			return wrapError(errPublishNacked)
		}
		return nil
	case <-ctx.Done():
		return wrapError(ctx.Err())
	}
}

// Package rabbitmq_stream implements the RabbitMQ Streams sink for hump.
package rabbitmq_stream

import (
	"context"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
)

// Process delivers a single WAL operation to a RabbitMQ Streams sink. It
// renders the message, ensures the producer for the rendered stream exists
// (running the per-(URL, stream) reconciler on first use), then publishes
// under the producer's serialization mutex and waits for the publisher
// confirmation. When capture is non-nil the rendered message's fields are
// recorded on it for the operations log.
func Process(ctx context.Context, cfg sinkcfg.RabbitMQStreamSink, operationContext *sink.Context, capture *sink.Capture) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	streamName, payload, headers, err := renderAll(cfg, operationContext)
	if err != nil {
		return wrapError(err)
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

	err = rp.prod.Send(msg)

	if capture != nil {
		capture.Target = streamName
		capture.SetPayload([]byte(payload))

		for name, value := range headers {
			capture.Set("header."+name, value)
		}
	}

	if err != nil {
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

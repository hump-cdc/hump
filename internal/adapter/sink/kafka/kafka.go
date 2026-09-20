package kafka

import (
	"context"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
)

// Process delivers a single WAL operation to a Kafka sink by acquiring (or
// reusing) the configured client, building a record from operationContext,
// and producing it synchronously with ctx. When capture is non-nil the built
// record's rendered fields are recorded on it for the operations log.
func Process(ctx context.Context, config sinkcfg.KafkaSink, operationContext *sink.SinkContext, capture *sink.Capture) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	client, err := getClient(config, operationContext)
	if err != nil {
		return wrapError(err)
	}

	message, err := buildMessage(config, operationContext)
	if err != nil {
		return wrapError(err)
	}

	err = client.ProduceSync(ctx, message).FirstErr()

	if capture != nil {
		capture.Target = message.Topic
		capture.SetPayload(message.Value)
		recordHeaders(message.Headers, capture)

		if len(message.Key) > 0 {
			capture.Set("key", string(message.Key))
		}
	}

	if err != nil {
		return wrapError(err)
	}

	return nil
}

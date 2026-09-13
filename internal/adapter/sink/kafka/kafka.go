package kafka

import (
	"context"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
)

// Process delivers a single WAL operation to a Kafka sink by acquiring (or
// reusing) the configured client, building a record from operationContext,
// and producing it synchronously with ctx. When streamEvent is non-nil the built
// record's rendered fields are recorded on it for the operations stream.
func Process(ctx context.Context, config sinkcfg.KafkaSink, operationContext *sink.SinkContext) error {
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

	if err := client.ProduceSync(ctx, message).FirstErr(); err != nil {
		return wrapError(err)
	}

	return nil
}

// Package rabbitmq implements the classic AMQP 0.9.1 sink for hump.
package rabbitmq

import (
	"context"

	"hump/internal/types/config"
	"hump/internal/types/sink"
	"hump/internal/types/stream"
)

// Process delivers a single WAL operation to a classic RabbitMQ AMQP sink.
// It acquires (or reuses) the cached connection, waits for it to be ready,
// renders the message, reconciles the exchange on first use, then publishes
// under the channel mutex and awaits the publisher confirmation (or a
// basic.return if the broker cannot route the mandatory message). When streamEvent is
// non-nil the rendered message's fields are recorded on it for the operations
// stream.
func Process(ctx context.Context, cfg config.RabbitMQSink, operationContext *sink.SinkContext, streamEvent *stream.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	rc, err := getClient(cfg, operationContext)
	if err != nil {
		return wrapError(err)
	}

	// Wait until the supervisor has a live channel — or bail with a transient
	// error if ctx fires first, so the retry layer keeps trying.
	select {
	case <-rc.readyChan():
	case <-ctx.Done():
		return wrapError(ctx.Err())
	}

	msg, err := buildMessage(cfg, operationContext)
	if err != nil {
		return wrapError(err)
	}

	if streamEvent != nil {
		streamEvent.Payload = string(msg.Publishing.Body)
		streamEvent.Headers = tableHeaders(msg.Publishing.Headers)
		streamEvent.Metadata = map[string]string{"routingKey": msg.RoutingKey}
	}

	if err := rc.reconciler.ensureDeclared(ctx, cfg.Exchange, msg.Exchange); err != nil {
		return wrapError(err)
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	ch, err := rc.healthyChannelLocked()
	if err != nil {
		return wrapError(err)
	}

	// Defensive: clear any stale confirm/return left by a prior aborted call.
	drainPending(rc.confirms, rc.returns)

	if err := ch.PublishWithContext(ctx, msg.Exchange, msg.RoutingKey, msg.Mandatory, false, msg.Publishing); err != nil {
		return wrapError(err)
	}

	return wrapError(awaitPublishResult(ctx, rc, msg))
}

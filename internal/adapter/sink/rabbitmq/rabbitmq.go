// Package rabbitmq implements the classic AMQP 0.9.1 sink for hump.
package rabbitmq

import (
	"context"
	"fmt"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
)

// Process delivers a single WAL operation to a classic RabbitMQ AMQP sink.
// It acquires (or reuses) the cached connection, waits for it to be ready,
// renders the message, reconciles the exchange on first use, then publishes
// under the channel mutex and awaits the publisher confirmation (or a
// basic.return if the broker cannot route the mandatory message). When capture
// is non-nil the rendered message's fields are recorded on it for the
// operations log.
func Process(ctx context.Context, cfg sinkcfg.RabbitMQSink, operationContext *sink.SinkContext, capture *sink.Capture) error {
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

	err = ch.PublishWithContext(ctx, msg.Exchange, msg.RoutingKey, msg.Mandatory, false, msg.Publishing)
	if err == nil {
		err = awaitPublishResult(ctx, rc, msg)
	}

	if capture != nil {
		capture.Target = msg.Exchange
		capture.SetPayload(msg.Publishing.Body)
		capture.Set("routingKey", msg.RoutingKey)

		for name, value := range msg.Publishing.Headers {
			capture.Set("header."+name, fmt.Sprint(value))
		}
	}

	return wrapError(err)
}

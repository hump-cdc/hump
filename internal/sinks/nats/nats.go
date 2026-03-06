// Package nats implements the NATS sink for hump.
package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"hump/internal/types/config"
	"hump/internal/types/sink"
)

// ProcessCore delivers a single WAL operation to a Core NATS sink by acquiring
// (or reusing) the configured client, building a message from
// operationContext, and publishing it with a synchronous flush.
func ProcessCore(ctx context.Context, cfg config.NatsCoreSink, operationContext *sink.SinkContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	nc, err := getClient(cfg.NatsCommon, operationContext)
	if err != nil {
		return wrapError(err)
	}

	msg, err := buildMessage(cfg.NatsCommon, operationContext)
	if err != nil {
		return wrapError(err)
	}

	if err := nc.conn.PublishMsg(msg); err != nil {
		return wrapError(err)
	}
	if err := nc.conn.FlushTimeout(flushTimeout(ctx)); err != nil {
		return wrapError(err)
	}

	return nil
}

// ProcessJetStream delivers a single WAL operation to a JetStream sink. It
// builds the message, applies the JetStream control headers, runs the
// per-(connection, stream) reconcile guard, and publishes asserting the
// expected stream.
func ProcessJetStream(ctx context.Context, cfg config.NatsJetStreamSink, operationContext *sink.SinkContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	nc, err := getClient(cfg.NatsCommon, operationContext)
	if err != nil {
		return wrapError(err)
	}

	if err := nc.reconciler.ensureReconciled(ctx, cfg); err != nil {
		return wrapError(err)
	}

	msg, err := buildMessage(cfg.NatsCommon, operationContext)
	if err != nil {
		return wrapError(err)
	}

	if err := applyJetStreamHeaders(msg.Header, cfg, operationContext); err != nil {
		return wrapError(err)
	}

	if _, err := nc.js.PublishMsg(ctx, msg, jetstream.WithExpectStream(cfg.Stream.Name)); err != nil {
		return wrapError(err)
	}

	return nil
}

// flushTimeout returns the remaining deadline on ctx, falling back to a small
// static default if ctx has no deadline.
func flushTimeout(ctx context.Context) time.Duration {
	if d, ok := ctx.Deadline(); ok {
		if rem := time.Until(d); rem > 0 {
			return rem
		}
	}
	return 5 * time.Second
}

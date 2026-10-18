// Package nats implements the NATS sink for hump.
package nats

import (
	"context"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
)

// ProcessCore delivers a single WAL operation to a Core NATS sink by acquiring
// (or reusing) the configured client, building a message from
// operationContext, and publishing it with a synchronous flush. When capture is
// non-nil the built message's rendered fields are recorded on it for the
// operations log.
func ProcessCore(ctx context.Context, cfg sinkcfg.NatsCoreSink, operationContext *sink.Context, capture *sink.Capture) error {
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

	err = nc.conn.PublishMsg(msg)
	if err == nil {
		err = nc.conn.FlushTimeout(flushTimeout(ctx))
	}

	captureMessage(capture, msg)

	if err != nil {
		return wrapError(err)
	}

	return nil
}

// ProcessJetStream delivers a single WAL operation to a JetStream sink. It
// builds the message, applies the JetStream control headers, runs the
// per-(connection, stream) reconcile guard, and publishes asserting the
// expected stream. When capture is non-nil the built message's rendered fields
// are recorded on it for the operations log.
func ProcessJetStream(ctx context.Context, cfg sinkcfg.NatsJetStreamSink, operationContext *sink.Context, capture *sink.Capture) error {
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

	ack, err := nc.js.PublishMsg(ctx, msg, jetstream.WithExpectStream(cfg.Stream.Name))

	captureMessage(capture, msg)
	if capture != nil {
		capture.Set("stream", cfg.Stream.Name)

		if ack != nil {
			capture.Set("sequence", strconv.FormatUint(ack.Sequence, 10))
		}
	}

	if err != nil {
		return wrapError(err)
	}

	return nil
}

// captureMessage records the subject, payload and headers of msg on capture,
// which is nil unless a client is watching the operations log.
func captureMessage(capture *sink.Capture, msg *nats.Msg) {
	if capture == nil {
		return
	}

	capture.Target = msg.Subject
	capture.SetPayload(msg.Data)

	for name := range msg.Header {
		capture.Set("header."+name, msg.Header.Get(name))
	}
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

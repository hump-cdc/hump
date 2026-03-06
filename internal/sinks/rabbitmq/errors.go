package rabbitmq

import (
	"context"
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"hump/internal/template"
	"hump/internal/types/errs"
)

// Sentinel errors classified by [wrapError].
var (
	// errReconnecting is returned by [Process] when the cached connection is
	// dropped and the supervisor goroutine has not yet re-established it.
	// Explicitly transient — must NOT be joined with [errs.ErrSinkPermanent].
	errReconnecting = errors.New("connection not ready: reconnecting")

	// errExchangeMissing is reported by the reconciler in `passive` mode when
	// the broker returns 404 NOT_FOUND for the declared exchange.
	errExchangeMissing = errors.New("exchange not found")

	// errExchangeMismatch is reported by the reconciler when the broker returns
	// 406 PRECONDITION_FAILED because an existing exchange has different settings
	// than the configured declaration.
	errExchangeMismatch = errors.New("exchange declaration mismatch")

	// errUnroutable is reported when the broker returns a basic.return for a
	// mandatory publish, indicating no queue is bound for the routing key.
	errUnroutable = errors.New("message unroutable")

	// errPublishNacked is reported when the broker NACKs the publish via
	// publisher confirms.
	errPublishNacked = errors.New("publish nacked by broker")
)

// wrapError returns a "rabbitmq" wrapped error, joining [errs.ErrSinkPermanent]
// for non-retriable errors: template render failures (shared across sinks) and
// the AMQP-specific failures classified by isPermanentRabbitError.
//
// errReconnecting is deliberately left in the transient branch so the retry
// layer keeps trying until the supervisor goroutine restores the connection.
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, template.ErrRender) || isPermanentRabbitError(err) {
		return fmt.Errorf("rabbitmq: %w", errors.Join(err, errs.ErrSinkPermanent))
	}
	return fmt.Errorf("rabbitmq: %w", err)
}

// isPermanentRabbitError reports whether err is a non-retriable AMQP error:
// the sink-specific sentinels, or an [*amqp.Error] carrying a channel-level
// code that won't clear by retrying (403/530/404/406/541).
//
// Connection-level codes (320 CONNECTION_FORCED, 501 FRAME_ERROR, 504
// CHANNEL_ERROR, 505 UNEXPECTED_FRAME), [amqp.ErrClosed], and ctx errors
// remain transient — they clear after the supervisor reconnects.
func isPermanentRabbitError(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, errExchangeMissing),
		errors.Is(err, errExchangeMismatch),
		errors.Is(err, errUnroutable),
		errors.Is(err, errPublishNacked):
		return true
	}

	if amqpErr, ok := errors.AsType[*amqp.Error](err); ok {
		switch amqpErr.Code {
		case amqp.AccessRefused, // 403
			amqp.NotAllowed,         // 530
			amqp.NotFound,           // 404
			amqp.PreconditionFailed, // 406
			amqp.InternalError:      // 541
			return true
		}
	}

	return false
}

// isTransientConnError reports whether err is a transport-level error from
// amqp091-go whose connection is not currently usable.
func isTransientConnError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, amqp.ErrClosed) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, errReconnecting) {
		return true
	}

	if amqpErr, ok := errors.AsType[*amqp.Error](err); ok {
		switch amqpErr.Code {
		case amqp.ConnectionForced, // 320
			amqp.FrameError,      // 501
			amqp.ChannelError,    // 504
			amqp.UnexpectedFrame: // 505
			return true
		}
	}

	return false
}

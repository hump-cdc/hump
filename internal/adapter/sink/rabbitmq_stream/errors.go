package rabbitmq_stream

import (
	"context"
	"errors"
	"fmt"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"

	"hump/internal/types/errs"
	"hump/pkg/template"
)

// Sentinel errors classified by [wrapError].
var (
	// errStreamMissing is reported in `passive` mode when the broker reports
	// the stream does not exist.
	errStreamMissing = errors.New("stream not found")

	// errStreamMismatch is reported by the reconciler when DeclareStream
	// fails because an existing stream has different retention than the
	// configured declaration.
	errStreamMismatch = errors.New("stream declaration mismatch")

	// errPublishNacked is reported when the producer's confirmation channel
	// reports the publish was not confirmed.
	errPublishNacked = errors.New("publish not confirmed by broker")
)

// wrapError returns a "rabbitmq-stream" wrapped error, joining
// [errs.ErrSinkPermanent] for non-retriable errors: template render failures
// (shared across sinks) and the stream-specific failures classified by
// isPermanentStreamError.
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, template.ErrRender) || isPermanentStreamError(err) {
		return fmt.Errorf("rabbitmq-stream: %w", errors.Join(err, errs.ErrSinkPermanent))
	}
	return fmt.Errorf("rabbitmq-stream: %w", err)
}

// isPermanentStreamError reports whether err is a non-retriable stream
// error: sink sentinels, auth failures, or precondition mismatches.
func isPermanentStreamError(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, errStreamMissing),
		errors.Is(err, errStreamMismatch),
		errors.Is(err, errPublishNacked),
		errors.Is(err, stream.AuthenticationFailure),
		errors.Is(err, stream.CodeAccessRefused),
		errors.Is(err, stream.PreconditionFailed):
		return true
	}

	return false
}

// isTransientStreamError reports whether err is a transport-level error from
// the stream client that will clear once the connection recovers. Used by
// the reconciler so transient probe failures are not cached as sticky.
func isTransientStreamError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, stream.ConnectionClosed) ||
		errors.Is(err, stream.AlreadyClosed) ||
		errors.Is(err, stream.StreamNotAvailable) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

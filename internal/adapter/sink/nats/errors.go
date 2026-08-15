package nats

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"hump/internal/port/sink"
	"hump/pkg/template"
)

// errCodeExpectedStreamMismatch is the JetStream API error code returned when
// a publish's Nats-Expected-Stream header does not match the routing stream.
// This is a transient error.
const errCodeExpectedStreamMismatch = 10060

var (
	errInvalidSubject = errors.New("invalid subject")
	errReconcile      = errors.New("reconcile")
)

// wrapError returns a "nats" wrapped error, joining [sink.ErrSinkPermanent]
// for non-retriable errors: template render failures (shared across sinks) and
// the NATS-specific failures classified by isPermanentNatsError.
func wrapError(err error) error {
	if errors.Is(err, template.ErrRender) || isPermanentNatsError(err) {
		return fmt.Errorf("nats: %w", errors.Join(err, sink.ErrSinkPermanent))
	}
	return fmt.Errorf("nats: %w", err)
}

// isTransientConnError reports whether err is a transport-level error from a
// NATS client whose connection is not currently usable: the broker is
// unreachable, the client is mid-reconnect, the underlying connection was
// closed, or the call ran out of time. Such errors clear once the client
// reconnects, so callers (e.g. the JetStream reconciler) must not cache them.
func isTransientConnError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, nats.ErrNoServers) ||
		errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrConnectionReconnecting) ||
		errors.Is(err, nats.ErrConnectionDraining) ||
		errors.Is(err, nats.ErrTimeout) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

// isPermanentNatsError reports whether err represents a NATS-specific failure
// that retrying will not fix: invalid subjects, reconcile failures, and the
// permanent NATS/JetStream API errors.
func isPermanentNatsError(err error) bool {
	if err == nil {
		return false
	}

	// Expected-stream mismatch is transient: check first to avoid being lumped
	// with the generic JetStream API-error catch below.
	var apiErr *jetstream.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode == errCodeExpectedStreamMismatch {
		return false
	}

	switch {
	case errors.Is(err, errInvalidSubject),
		errors.Is(err, errReconcile),
		errors.Is(err, nats.ErrMaxPayload),
		errors.Is(err, nats.ErrAuthorization),
		errors.Is(err, nats.ErrAuthExpired),
		errors.Is(err, nats.ErrPermissionViolation),
		errors.Is(err, jetstream.ErrStreamNotFound):
		return true
	}

	return false
}

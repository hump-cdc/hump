package resilience

import "errors"

var (
	// ErrRetryAborted wraps the last attempt's error when WithRetry gave up early
	// because its stop channel (shutdown) was closed.
	ErrRetryAborted = errors.New("retry aborted by shutdown")
)

package resilience

import (
	"context"
	"time"
)

// WithTimeout wraps fn so it is cancelled after timeout, returning a wrapped
// context.DeadlineExceeded error when the deadline is hit. A non-positive
// timeout disables the wrapper, and fn is returned as is.
func WithTimeout[T any](fn Handler[T], timeout time.Duration) Handler[T] {
	if timeout <= 0 {
		return fn
	}

	return func(ctx context.Context, item T) error {
		c, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return fn(c, item)
	}
}

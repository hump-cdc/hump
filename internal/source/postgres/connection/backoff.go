package connection

import (
	"context"
	"time"
)

const (
	// baseDelay is the wait before the first retry; subsequent attempts double it.
	baseDelay = 100 * time.Millisecond
	// maxDelay caps the per-attempt wait regardless of the attempt index.
	maxDelay = 5 * time.Second
)

// backoff blocks for an exponentially growing duration based on attempt,
// clamped to maxDelay. It returns ctx.Err() if ctx is cancelled while waiting.
func backoff(ctx context.Context, attempt int) error {
	delay := baseDelay << attempt
	if delay <= 0 || delay > maxDelay {
		delay = maxDelay
	}

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

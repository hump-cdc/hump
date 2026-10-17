// Package resilience wraps a Handler with timeouts, retries with backoff,
// outcome hooks and a failure mode.
package resilience

import "context"

// Handler is a unit of work over a single item.
type Handler[T any] func(ctx context.Context, item T) error

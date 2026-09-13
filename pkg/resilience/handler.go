package resilience

import "context"

// Handler is a unit of work over a single item.
type Handler[T any] func(ctx context.Context, item T) error

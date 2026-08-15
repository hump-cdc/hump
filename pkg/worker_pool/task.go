package workerpool

import "context"

type task[T any] struct {
	ctx     context.Context
	fn      func(ctx context.Context, payload T) error
	payload T
}

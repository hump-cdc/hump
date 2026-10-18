package resilience

import "context"

// StatusHooks receives the terminal outcome of WithStatus.
type StatusHooks interface {
	// OnSuccess is called when the wrapped handler returned no error.
	OnSuccess()
	// OnFailure is called when the wrapped handler returned an error.
	OnFailure()
}

type noopStatusHooks struct{}

func (noopStatusHooks) OnSuccess() {}
func (noopStatusHooks) OnFailure() {}

// WithStatus wraps fn so exactly one of hooks.OnSuccess/OnFailure is called
// after every invocation, based on the returned error.
func WithStatus[T any](fn Handler[T], hooks StatusHooks) Handler[T] {
	if hooks == nil {
		hooks = noopStatusHooks{}
	}

	return func(ctx context.Context, item T) error {
		err := fn(ctx, item)
		if err == nil {
			hooks.OnSuccess()
		} else {
			hooks.OnFailure()
		}
		return err
	}
}

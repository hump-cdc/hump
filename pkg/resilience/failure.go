package resilience

import "context"

// FailureMode selects what happens once every retry attempt is exhausted.
type FailureMode string

const (
	// FailureModeCrash reports the failure to the caller.
	FailureModeCrash FailureMode = "crash"
	// FailureModeSkip drops the failed item and carries on.
	FailureModeSkip FailureMode = "skip"
)

// SkipHooks receives the errors WithFailureMode discards.
type SkipHooks interface {
	// OnSkip is called with the error of a call whose failure was swallowed.
	OnSkip(err error)
}

type noopSkipHooks struct{}

func (noopSkipHooks) OnSkip(error) {}

// WithFailureMode wraps fn so a failing call is dropped instead of reported
// when mode is FailureModeSkip: the error goes to hooks and nil is returned.
// Any other mode disables the wrapper, and fn is returned as is.
func WithFailureMode[T any](fn Handler[T], mode FailureMode, hooks SkipHooks) Handler[T] {
	if mode != FailureModeSkip {
		return fn
	}

	if hooks == nil {
		hooks = noopSkipHooks{}
	}

	return func(ctx context.Context, item T) error {
		err := fn(ctx, item)
		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return err
		}

		hooks.OnSkip(err)

		return nil
	}
}

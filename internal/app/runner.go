package app

import (
	"context"
	"errors"
	"hump/internal/app/shutdown"
	"syscall"
	"time"
)

type runner struct {
	context context.Context
	cancel  context.CancelCauseFunc
	release context.CancelFunc
}

// Creates a runner whose context is cancelled on SIGINT/SIGTERM or when
// any function passed to Run returns an error. The returned CancelFunc releases
// the OS signal hook and should be deferred by the caller.
func New(ctx context.Context) (*runner, context.CancelFunc) {
	ctx, release := signals(ctx, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancelCause(ctx)

	return &runner{
		release: release,
		context: ctx,
		cancel:  cancel,
	}, release
}

// Run launches each function in its own goroutine. The first function to return
// a non-nil error cancels the runner's context with that error as the cause,
// triggering shutdown for all the others.
func (r *runner) Run(fns ...func(context.Context) error) {
	for _, fn := range fns {
		go func() {
			if err := fn(r.context); err != nil {
				r.cancel(err)
			}
		}()
	}
}

// Wait blocks until the runner's context is done (OS signal or error),
// then releases the signal hook, runs registered shutdown hooks bounded by
// timeout, and returns the cause.
func (r *runner) Wait() error {
	// wait OS signals or error from any runner
	<-r.context.Done()

	// release OS signals hook
	r.release()

	// shutdown services
	shutdown.Shutdown(5 * time.Second)

	// extract error
	if err := context.Cause(r.context); err != nil && !errors.Is(err, errSignalInterrupt) && !errors.Is(err, errSignalTerminate) {
		return err
	}

	return nil
}

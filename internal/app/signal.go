package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

var (
	// SIGINT is received.
	errSignalInterrupt = errors.New("interrupt signal")
	// SIGTERM is received.
	errSignalTerminate = errors.New("terminate signal")
)

// signals returns a child context that is cancelled when any of the supplied
// OS signals is delivered, together with a cancel function that stops the
// signal handler and releases its resources.
//
// The cancellation cause exposed via [context.Cause] identifies the trigger:
// [errSignalInterrupt] for SIGINT, [errSignalTerminate] for SIGTERM, or an
// error whose message is the signal's String() for any other registered
// signal. Invoking the returned cancel function cancels with a nil cause.
//
// The returned cancel function must be called to release the signal
// subscription and let the watcher goroutine exit.
func signals(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(ctx)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, sig...)
	if ctx.Err() == nil {
		go func() {
			select {
			case s := <-signalChannel:
				switch s {
				case syscall.SIGINT:
					cancel(errSignalInterrupt)
				case syscall.SIGTERM:
					cancel(errSignalTerminate)
				default:
					cancel(errors.New(s.String()))
				}
			case <-ctx.Done():
			}
		}()
	}

	return ctx, func() {
		cancel(nil)
		signal.Stop(signalChannel)
	}
}

package resilience

import (
	"context"
	"sync"
	"time"
)

// RetryHooks observes each attempt made by WithRetry.
type RetryHooks interface {
	// OnDuration is called after every attempt with how long it took,
	// whether it succeeded or failed.
	OnDuration(elapsed time.Duration)
	// OnRetry is called once per retry, after the backoff pause and before
	// the next attempt starts.
	OnRetry()
}

type noopRetryHooks struct{}

func (noopRetryHooks) OnDuration(time.Duration) {}
func (noopRetryHooks) OnRetry()                 {}

// timerPool reuses backoff timers so a retrying call does not allocate one.
// Reusing a timer without draining its channel is safe as of Go 1.23: after
// Stop or Reset returns, no stale value can be received from it. This relies
// on the go directive in go.mod staying at 1.23 or above.
var timerPool = sync.Pool{
	New: func() any {
		// The duration is arbitrary: every caller resets the timer before
		// waiting on it, so it only has to come out of the pool disarmed.
		t := time.NewTimer(time.Hour)
		t.Stop()

		return t
	},
}

// pause waits for d or until ctx is done, whichever comes first, returning
// ctx.Err() when the wait was cut short. The timer it waits on comes from
// timerPool and goes back as soon as the wait is over.
func pause(ctx context.Context, d time.Duration) error {
	timer := timerPool.Get().(*time.Timer)
	timer.Reset(d)

	select {
	case <-ctx.Done():
		timer.Stop()
		timerPool.Put(timer)

		return ctx.Err()
	case <-timer.C:
		timerPool.Put(timer)

		return nil
	}
}

// WithRetry wraps fn so a failing call is retried up to attempts times, pausing
// between attempts per backoff and reporting each one to hooks.
func WithRetry[T any](fn Handler[T], attempts uint, backoff Backoff, hooks RetryHooks) Handler[T] {
	if attempts == 0 {
		attempts = 1
	}

	if hooks == nil {
		hooks = noopRetryHooks{}
	}

	delay := backoff.DelayFunc()

	return func(ctx context.Context, item T) error {
		var err error
		for i := range attempts {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			start := time.Now()
			err = fn(ctx, item)
			hooks.OnDuration(time.Since(start))

			if err == nil {
				return nil
			}

			if i+1 < attempts {
				if d := delay(i + 1); d > 0 {
					if ctxErr := pause(ctx, d); ctxErr != nil {
						return ctxErr
					}
				}

				hooks.OnRetry()
			}
		}

		return err
	}
}

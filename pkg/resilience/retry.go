package resilience

import (
	"context"
	"errors"
	"fmt"
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

// WithRetry wraps fn so a failing call is retried up to attempts times, pausing
// between attempts per backoff and reporting each one to hooks. A stop closing
// mid-pause triggers one immediate last attempt; a stop already closed when an
// attempt fails ends the retries. A nil stop never closes.
func WithRetry[T any](fn Handler[T], attempts uint, backoff Backoff, hooks RetryHooks, stop <-chan struct{}) Handler[T] {
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
				if pauseErr := pause(ctx, stop, delay(i+1)); pauseErr != nil {
					if errors.Is(pauseErr, ErrRetryAborted) {
						return fmt.Errorf("%w: %w", ErrRetryAborted, err)
					}

					return pauseErr
				}

				hooks.OnRetry()
			}
		}

		return err
	}
}

// pause waits for d, returning ErrRetryAborted when stop is already closed and
// ctx.Err() when ctx is done first. A stop closing mid-wait cuts it short with
// nil, so the caller retries at once. The timer comes from timerPool.
func pause(ctx context.Context, stop <-chan struct{}, d time.Duration) error {
	select {
	case <-stop:
		return ErrRetryAborted
	default:
	}

	if d <= 0 {
		return nil
	}

	timer := timerPool.Get().(*time.Timer)
	timer.Reset(d)

	select {
	case <-stop:
		timer.Stop()
		timerPool.Put(timer)

		return nil
	case <-ctx.Done():
		timer.Stop()
		timerPool.Put(timer)

		return ctx.Err()
	case <-timer.C:
		timerPool.Put(timer)

		return nil
	}
}

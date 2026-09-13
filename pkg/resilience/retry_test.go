package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type countingHooks struct {
	durations int
	retries   int
}

func (h *countingHooks) OnDuration(time.Duration) { h.durations++ }
func (h *countingHooks) OnRetry()                 { h.retries++ }

func TestWithRetry_nilHooks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		calls := 0
		fn := func(context.Context, int) error {
			calls++
			return nil
		}
		handler := WithRetry(fn, 1, Backoff{}, nil, nil)

		if err := handler(context.Background(), 1); err != nil {
			t.Fatalf("got error %v, want nil", err)
		}
		if calls != 1 {
			t.Fatalf("fn called %d times, want 1", calls)
		}
	})

	t.Run("failure cancelled during pause", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		h := WithRetry(func(context.Context, int) error {
			cancel()
			return errors.New("boom")
		}, 3, Backoff{Type: BackoffTypeLinear}, nil, nil)

		if err := h(ctx, 1); !errors.Is(err, context.Canceled) {
			t.Fatalf("got error %v, want context.Canceled", err)
		}
	})
}

func TestWithRetry_backoffApplied(t *testing.T) {
	hooks := &countingHooks{}
	wantErr := errors.New("boom")

	h := WithRetry(func(context.Context, int) error {
		return wantErr
	}, 3, Backoff{
		Type:      BackoffTypeLinear,
		Initial:   time.Millisecond,
		Increment: time.Millisecond,
		Jitter:    JitterModeNone,
	}, hooks, nil)

	start := time.Now()
	err := h(context.Background(), 1)
	elapsed := time.Since(start)

	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if want := 3 * time.Millisecond; elapsed < want {
		t.Fatalf("elapsed %v, want at least %v", elapsed, want)
	}
	if hooks.durations != 3 {
		t.Fatalf("OnDuration called %d times, want 3", hooks.durations)
	}
	if hooks.retries != 2 {
		t.Fatalf("OnRetry called %d times, want 2", hooks.retries)
	}
}

func TestWithRetry_zeroDelayStillReportsRetry(t *testing.T) {
	hooks := &countingHooks{}
	wantErr := errors.New("boom")

	h := WithRetry(func(context.Context, int) error {
		return wantErr
	}, 3, Backoff{}, hooks, nil)

	if err := h(context.Background(), 1); !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if hooks.durations != 3 {
		t.Fatalf("OnDuration called %d times, want 3", hooks.durations)
	}
	if hooks.retries != 2 {
		t.Fatalf("OnRetry called %d times, want 2", hooks.retries)
	}
}

func TestWithRetry_zeroAttemptsRunsOnce(t *testing.T) {
	hooks := &countingHooks{}
	wantErr := errors.New("boom")

	calls := 0
	h := WithRetry(func(context.Context, int) error {
		calls++
		return wantErr
	}, 0, Backoff{}, hooks, nil)

	if err := h(context.Background(), 1); !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
	if hooks.durations != 1 {
		t.Fatalf("OnDuration called %d times, want 1", hooks.durations)
	}
	if hooks.retries != 0 {
		t.Fatalf("OnRetry called %d times, want 0", hooks.retries)
	}
}

func TestWithRetry_cancelledDuringBackoffPause(t *testing.T) {
	hooks := &countingHooks{}
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	// The pause is long enough that only the cancellation can end it.
	h := WithRetry(func(context.Context, int) error {
		calls++
		cancel()
		return errors.New("boom")
	}, 3, Backoff{Type: BackoffTypeLinear, Initial: time.Hour}, hooks, nil)

	if err := h(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
	if hooks.durations != 1 {
		t.Fatalf("OnDuration called %d times, want 1", hooks.durations)
	}
	if hooks.retries != 0 {
		t.Fatalf("OnRetry called %d times, want 0", hooks.retries)
	}
}

func TestWithRetry_pooledTimerNotStale(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cancelled := WithRetry(func(context.Context, int) error {
		cancel()
		return errors.New("boom")
	}, 3, Backoff{Type: BackoffTypeLinear, Initial: time.Hour}, nil, nil)

	if err := cancelled(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}

	const pause = 20 * time.Millisecond
	wantErr := errors.New("boom")

	h := WithRetry(func(context.Context, int) error {
		return wantErr
	}, 2, Backoff{
		Type:    BackoffTypeLinear,
		Initial: pause,
		Jitter:  JitterModeNone,
	}, nil, nil)

	start := time.Now()
	err := h(context.Background(), 1)
	elapsed := time.Since(start)

	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if elapsed < pause {
		t.Fatalf("elapsed %v, want at least %v: the pause was cut short by a stale timer", elapsed, pause)
	}
}

func TestWithRetry_concurrentPauses(t *testing.T) {
	const goroutines = 50

	wantErr := errors.New("boom")
	h := WithRetry(func(context.Context, int) error {
		return wantErr
	}, 3, Backoff{
		Type:    BackoffTypeLinear,
		Initial: time.Millisecond,
		Jitter:  JitterModeNone,
	}, nil, nil)

	errs := make(chan error, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			errs <- h(context.Background(), 1)
		}()
	}
	wg.Wait()
	close(errs)

	got := 0
	for err := range errs {
		if !errors.Is(err, wantErr) {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
		got++
	}
	if got != goroutines {
		t.Fatalf("%d calls returned, want %d", got, goroutines)
	}
}

func TestWithRetry_stopBeforeFirstAttemptStillRunsIt(t *testing.T) {
	stop := make(chan struct{})
	close(stop)

	calls := 0
	h := WithRetry(func(context.Context, int) error {
		calls++
		return nil
	}, 3, Backoff{}, nil, stop)

	if err := h(context.Background(), 1); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
}

func TestWithRetry_stopAbortsRetries(t *testing.T) {
	hooks := &countingHooks{}
	wantErr := errors.New("boom")
	stop := make(chan struct{})

	calls := 0
	h := WithRetry(func(context.Context, int) error {
		calls++
		close(stop)
		return wantErr
	}, 3, Backoff{}, hooks, stop)

	err := h(context.Background(), 1)
	if !errors.Is(err, ErrRetryAborted) || !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want ErrRetryAborted wrapping %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
	if hooks.durations != 1 {
		t.Fatalf("OnDuration called %d times, want 1", hooks.durations)
	}
	if hooks.retries != 0 {
		t.Fatalf("OnRetry called %d times, want 0", hooks.retries)
	}
}

func TestWithRetry_stopDuringBackoffPauseRetriesOnce(t *testing.T) {
	hooks := &countingHooks{}
	wantErr := errors.New("boom")
	stop := make(chan struct{})

	calls := 0
	// The pause is long enough that only the stop can end it.
	h := WithRetry(func(context.Context, int) error {
		calls++
		if calls == 1 {
			time.AfterFunc(10*time.Millisecond, func() { close(stop) })
		}
		return wantErr
	}, 5, Backoff{Type: BackoffTypeLinear, Initial: time.Hour}, hooks, stop)

	err := h(context.Background(), 1)
	if !errors.Is(err, ErrRetryAborted) || !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want ErrRetryAborted wrapping %v", err, wantErr)
	}
	if calls != 2 {
		t.Fatalf("fn called %d times, want 2", calls)
	}
	if hooks.retries != 1 {
		t.Fatalf("OnRetry called %d times, want 1", hooks.retries)
	}
}

func TestWithRetry_stopDuringBackoffPauseRetrySucceeds(t *testing.T) {
	stop := make(chan struct{})

	calls := 0
	h := WithRetry(func(context.Context, int) error {
		calls++
		if calls == 1 {
			time.AfterFunc(10*time.Millisecond, func() { close(stop) })
			return errors.New("boom")
		}
		return nil
	}, 5, Backoff{Type: BackoffTypeLinear, Initial: time.Hour}, nil, stop)

	if err := h(context.Background(), 1); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if calls != 2 {
		t.Fatalf("fn called %d times, want 2", calls)
	}
}

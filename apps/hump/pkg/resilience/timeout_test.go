package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithTimeout_passthrough(t *testing.T) {
	wantErr := errors.New("boom")

	tests := map[string]error{
		"success": nil,
		"failure": wantErr,
	}

	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			h := WithTimeout(func(context.Context, int) error { return want }, time.Minute)

			if err := h(context.Background(), 1); !errors.Is(err, want) {
				t.Fatalf("got error %v, want %v", err, want)
			}
		})
	}
}

func TestWithTimeout_deadlineSet(t *testing.T) {
	const timeout = time.Minute

	var (
		deadline time.Time
		ok       bool
	)
	h := WithTimeout(func(ctx context.Context, _ int) error {
		deadline, ok = ctx.Deadline()
		return nil
	}, timeout)

	if err := h(context.Background(), 1); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !ok {
		t.Fatal("no deadline set on context, want one")
	}
	if left := time.Until(deadline); left <= 0 || left > timeout {
		t.Fatalf("deadline in %v, want within (0, %v]", left, timeout)
	}
}

func TestWithTimeout_deadlineExceeded(t *testing.T) {
	h := WithTimeout(func(ctx context.Context, _ int) error {
		<-ctx.Done()
		return ctx.Err()
	}, time.Millisecond)

	if err := h(context.Background(), 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got error %v, want context.DeadlineExceeded", err)
	}
}

func TestWithTimeout_parentCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := WithTimeout(func(ctx context.Context, _ int) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	}, time.Minute)

	if err := h(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
}

func TestWithTimeout_derivedContextCancelledOnReturn(t *testing.T) {
	var inner context.Context

	h := WithTimeout(func(ctx context.Context, _ int) error {
		inner = ctx
		return nil
	}, time.Minute)

	if err := h(context.Background(), 1); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if err := inner.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("derived context error %v, want context.Canceled", err)
	}
}

func TestWithTimeout_sleepPastDeadline(t *testing.T) {
	const (
		timeout = time.Millisecond
		sleep   = 20 * time.Millisecond
	)

	h := WithTimeout(func(ctx context.Context, _ int) error {
		<-time.After(sleep)
		return ctx.Err()
	}, timeout)

	start := time.Now()
	err := h(context.Background(), 1)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got error %v, want context.DeadlineExceeded", err)
	}
	// The timeout does not preempt fn: the handler returns only once fn does.
	if elapsed < sleep {
		t.Fatalf("elapsed %v, want at least %v", elapsed, sleep)
	}
}

func TestWithTimeout_nonPositiveTimeout(t *testing.T) {
	for name, timeout := range map[string]time.Duration{"zero": 0, "negative": -time.Second} {
		t.Run(name, func(t *testing.T) {
			var (
				inner context.Context
				ok    bool
			)
			h := WithTimeout(func(ctx context.Context, _ int) error {
				_, ok = ctx.Deadline()
				inner = ctx

				return nil
			}, timeout)

			if err := h(context.Background(), 1); err != nil {
				t.Fatalf("got error %v, want nil", err)
			}
			if ok {
				t.Fatal("deadline set on context, want none")
			}
			// The wrapper is disabled, so there is no derived context to
			// cancel once the handler returns.
			if err := inner.Err(); err != nil {
				t.Fatalf("context error %v, want nil", err)
			}
		})
	}
}

package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

type noopHooks struct {
	durations int
	retries   int
	successes int
	failures  int
}

func (h *noopHooks) OnDuration(time.Duration) { h.durations++ }
func (h *noopHooks) OnRetry()                 { h.retries++ }
func (h *noopHooks) OnSuccess()               { h.successes++ }
func (h *noopHooks) OnFailure()               { h.failures++ }

// BenchmarkResilience measures the hot-path case with and without retries.
// The retry path itself is allocation-free: the backoff timer comes from a
// pool. WithTimeout derives a context per attempt, so the reported
// allocations are expected to scale with the attempt count.
func BenchmarkResilience(b *testing.B) {
	tests := []struct {
		name    string
		retries int
	}{
		{name: "first success", retries: 0},
		{name: "1 attempt", retries: 1},
		{name: "2 attempts", retries: 2},
		{name: "4 attempts", retries: 4},
		{name: "6 attempts", retries: 6},
		{name: "12 attempts", retries: 12},
	}

	err := errors.New("expected err")
	hooks := &noopHooks{}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			attempt := 0
			fn := func(context.Context, int) error {
				attempt++
				if attempt < tt.retries {
					return err
				}
				return nil
			}
			// fn = WithTimeout(fn, 1*time.Millisecond)
			fn = WithRetry(fn, uint(tt.retries), Backoff{
				Type:       BackoffTypeExponential,
				Initial:    time.Millisecond,
				Max:        30 * time.Second,
				Multiplier: 2,
				Jitter:     JitterModeFull,
			}, hooks)
			fn = WithStatus(fn, hooks)

			ctx := context.Background()

			b.ReportAllocs()
			for b.Loop() {
				attempt = 0
				if err := fn(ctx, 1); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

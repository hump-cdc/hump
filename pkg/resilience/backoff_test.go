package resilience

import (
	"math"
	"testing"
	"time"
)

func TestBackoffDelayFunc(t *testing.T) {
	tests := map[string]struct {
		backoff     Backoff
		attempts    []uint          // attempts to evaluate, defaults to 1, 2, 3
		want        []time.Duration // exact delay per attempt
		wantAtLeast time.Duration
		wantBelow   time.Duration
		repeat      int // evaluations per attempt, >1 for random draws
	}{
		"exponential": {
			backoff: Backoff{
				Type:       BackoffTypeExponential,
				Initial:    100 * time.Millisecond,
				Multiplier: 2,
				Jitter:     JitterModeNone,
			},
			want: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond},
		},
		"exponential with default multiplier": {
			backoff: Backoff{
				Type:    BackoffTypeExponential,
				Initial: 100 * time.Millisecond,
			},
			want: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond},
		},
		"unknown type falls back to linear": {
			backoff: Backoff{
				Type:      BackoffType("parabolic"),
				Initial:   100 * time.Millisecond,
				Increment: 100 * time.Millisecond,
			},
			want: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond},
		},
		"linear": {
			backoff: Backoff{
				Type:      BackoffTypeLinear,
				Initial:   500 * time.Millisecond,
				Increment: 500 * time.Millisecond,
				Jitter:    JitterModeNone,
			},
			want: []time.Duration{500 * time.Millisecond, 1000 * time.Millisecond, 1500 * time.Millisecond},
		},
		"linear without increment is flat": {
			backoff: Backoff{
				Type:    BackoffTypeLinear,
				Initial: 500 * time.Millisecond,
			},
			want: []time.Duration{500 * time.Millisecond, 500 * time.Millisecond, 500 * time.Millisecond},
		},
		"max caps the curve": {
			backoff: Backoff{
				Type:       BackoffTypeExponential,
				Initial:    100 * time.Millisecond,
				Max:        150 * time.Millisecond,
				Multiplier: 2,
			},
			want: []time.Duration{100 * time.Millisecond, 150 * time.Millisecond, 150 * time.Millisecond},
		},
		"zero value never pauses": {
			backoff: Backoff{},
			want:    []time.Duration{0, 0, 0},
		},
		"full jitter stays within the curve": {
			backoff: Backoff{
				Type:       BackoffTypeExponential,
				Initial:    100 * time.Millisecond,
				Multiplier: 2,
				Jitter:     JitterModeFull,
			},
			attempts:  []uint{3},
			wantBelow: 400 * time.Millisecond, // Initial × Multiplier^2
			repeat:    1000,
		},
		"exponential capped at a high attempt": {
			backoff: Backoff{
				Type:    BackoffTypeExponential,
				Initial: time.Second,
				Max:     30 * time.Second,
			},
			attempts: []uint{1000},
			want:     []time.Duration{30 * time.Second},
		},
		"exponential uncapped at a high attempt": {
			backoff: Backoff{
				Type:    BackoffTypeExponential,
				Initial: time.Second,
			},
			attempts:    []uint{1000},
			wantAtLeast: 1 * time.Second,
		},
		"linear capped at a high attempt": {
			backoff: Backoff{
				Type:      BackoffTypeLinear,
				Initial:   time.Second,
				Increment: time.Second,
				Max:       30 * time.Second,
			},
			attempts: []uint{1 << 40},
			want:     []time.Duration{30 * time.Second},
		},
		"linear uncapped at a high attempt": {
			backoff: Backoff{
				Type:      BackoffTypeLinear,
				Initial:   time.Second,
				Increment: time.Second,
			},
			attempts:    []uint{1 << 40},
			wantAtLeast: 1 * time.Second,
		},
		"linear with huge increment": {
			backoff: Backoff{
				Type:      BackoffTypeLinear,
				Initial:   time.Second,
				Increment: time.Duration(math.MaxInt64 / 4),
			},
			attempts:    []uint{64},
			wantAtLeast: 1 * time.Second,
		},
		// The curves clamp to Max on their own; this reaches the final clamp by
		// overflowing the linear curve with a negative Initial (raw value 101ns).
		"linear overflows past max": {
			backoff: Backoff{
				Type:      BackoffTypeLinear,
				Initial:   math.MinInt64 + 1,
				Increment: 2,
				Max:       100,
			},
			attempts: []uint{1<<62 + 51},
			want:     []time.Duration{100},
		},
		"linear initial above max": {
			backoff: Backoff{
				Type:      BackoffTypeLinear,
				Initial:   time.Minute,
				Increment: time.Second,
				Max:       time.Second},
			attempts: []uint{1 << 40},
			want:     []time.Duration{time.Second},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			delay := tt.backoff.DelayFunc()

			attempts := tt.attempts
			if attempts == nil {
				attempts = []uint{1, 2, 3}
			}
			repeat := max(tt.repeat, 1)

			for i, attempt := range attempts {
				for range repeat {
					got := delay(attempt)

					if tt.want != nil {
						if got != tt.want[i] {
							t.Fatalf("delay(%d) = %v, want %v", attempt, got, tt.want[i])
						}
						continue
					}
					if got < tt.wantAtLeast {
						t.Fatalf("delay(%d) = %v, want at least %v", attempt, got, tt.wantAtLeast)
					}
					if tt.wantBelow > 0 && got >= tt.wantBelow {
						t.Fatalf("delay(%d) = %v, want below %v", attempt, got, tt.wantBelow)
					}
				}
			}
		})
	}
}

package resilience

import (
	"math"
	"math/rand/v2"
	"time"
)

// BackoffType selects the backoff curve between retry attempts.
type BackoffType string

const (
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeExponential BackoffType = "exponential"
)

// JitterMode selects the randomization applied on top of the backoff delay.
type JitterMode string

const (
	JitterModeFull JitterMode = "full"
	JitterModeNone JitterMode = "none"
)

// Backoff describes how the delay between retry attempts grows over time.
// Which fields apply depends on Type, as noted on each one.
type Backoff struct {
	Type       BackoffType
	Initial    time.Duration
	Max        time.Duration
	Increment  time.Duration // linear only
	Multiplier float64       // exponential only
	Jitter     JitterMode    // exponential only
}

// DelayFunc builds the delay function for this backoff, mapping a 1-based
// attempt number to the wait before it, capped by Max and jittered per Jitter.
// An unset or unknown Type falls back to the linear curve.
func (b Backoff) DelayFunc() func(attempt uint) time.Duration {
	// Growth stops here instead of running past what a Duration holds.
	ceiling := time.Duration(math.MaxInt64)
	if b.Max > 0 {
		ceiling = b.Max
	}

	var curve func(attempt uint) time.Duration

	switch b.Type {
	case BackoffTypeExponential:
		initial := float64(b.Initial)
		multiplier := b.Multiplier
		if multiplier <= 0 {
			multiplier = 2
		}

		ceilingF := float64(ceiling)
		curve = func(attempt uint) time.Duration {
			d := initial
			for range attempt - 1 {
				if d >= ceilingF {
					return ceiling
				}
				d *= multiplier
			}
			if d >= ceilingF {
				return ceiling
			}

			return time.Duration(d)
		}
	default:
		initial, increment := b.Initial, b.Increment
		curve = func(attempt uint) time.Duration {
			if initial >= ceiling || increment <= 0 {
				return min(initial, ceiling)
			}

			steps := uint64(attempt - 1)
			if steps > uint64((ceiling-initial)/increment) {
				return ceiling
			}

			return initial + increment*time.Duration(steps)
		}
	}

	return func(attempt uint) time.Duration {
		delay := curve(attempt)

		if b.Max > 0 && delay > b.Max {
			delay = b.Max
		}

		if b.Jitter == JitterModeFull && delay > 0 {
			// [0, delay) jitter
			delay = time.Duration(rand.Int64N(int64(delay)))
		}

		return delay
	}
}

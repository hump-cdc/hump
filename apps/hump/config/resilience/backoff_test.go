package resilience

import (
	"testing"
	"time"

	"hump/pkg/resilience"

	"github.com/goccy/go-yaml"
)

func TestBackoffDefaults(t *testing.T) {
	var b Backoff
	if err := yaml.Unmarshal([]byte("{}"), &b); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	want := Backoff{
		Type:       resilience.BackoffTypeExponential,
		Initial:    time.Second,
		Max:        30 * time.Second,
		Multiplier: 2,
		Jitter:     resilience.JitterModeFull,
	}
	if b != want {
		t.Fatalf("Backoff = %+v, want %+v", b, want)
	}
}

func TestBackoffExplicitValuesKept(t *testing.T) {
	src := "type: linear\ninitial: 2s\nmax: 1m\nincrement: 3s\nmultiplier: 1.5\njitter: none\n"

	var b Backoff
	if err := yaml.Unmarshal([]byte(src), &b); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	want := Backoff{
		Type:       resilience.BackoffTypeLinear,
		Initial:    2 * time.Second,
		Max:        time.Minute,
		Increment:  3 * time.Second,
		Multiplier: 1.5,
		Jitter:     resilience.JitterModeNone,
	}
	if b != want {
		t.Fatalf("Backoff = %+v, want %+v", b, want)
	}
}

func TestBackoffInvalidRejected(t *testing.T) {
	var b Backoff
	if err := yaml.Unmarshal([]byte("type: bogus\n"), &b); err == nil {
		t.Fatalf("expected validation error for unknown backoff type")
	}
}

func TestBackoffPolicy(t *testing.T) {
	b := Backoff{
		Type:       resilience.BackoffTypeLinear,
		Initial:    time.Second,
		Max:        time.Minute,
		Increment:  2 * time.Second,
		Multiplier: 3,
		Jitter:     resilience.JitterModeNone,
	}

	got := b.Policy()
	want := resilience.Backoff{
		Type:       b.Type,
		Initial:    b.Initial,
		Max:        b.Max,
		Increment:  b.Increment,
		Multiplier: b.Multiplier,
		Jitter:     b.Jitter,
	}
	if got != want {
		t.Fatalf("Policy = %+v, want %+v", got, want)
	}
}

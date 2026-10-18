package resilience

import (
	"testing"
	"time"

	"hump/pkg/resilience"

	"github.com/goccy/go-yaml"
)

func TestRetryDefaults(t *testing.T) {
	var r Retry
	if err := yaml.Unmarshal([]byte("{}"), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if r.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", r.MaxAttempts)
	}
	if r.AttemptTimeout != 10*time.Second {
		t.Errorf("AttemptTimeout = %v, want 10s", r.AttemptTimeout)
	}
	if r.Backoff.Type != resilience.BackoffTypeExponential {
		t.Errorf("Backoff.Type = %q, want %q", r.Backoff.Type, resilience.BackoffTypeExponential)
	}
}

func TestRetryExplicitValuesKept(t *testing.T) {
	src := "maxAttempts: 3\nattemptTimeout: 2s\nbackoff:\n  type: linear\n  increment: 1s\n"

	var r Retry
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if r.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", r.MaxAttempts)
	}
	if r.AttemptTimeout != 2*time.Second {
		t.Errorf("AttemptTimeout = %v, want 2s", r.AttemptTimeout)
	}
	if r.Backoff.Type != resilience.BackoffTypeLinear || r.Backoff.Increment != time.Second {
		t.Errorf("Backoff = %+v, want linear with 1s increment", r.Backoff)
	}
}

func TestRetryInvalidRejected(t *testing.T) {
	var r Retry
	if err := yaml.Unmarshal([]byte("maxAttempts: 3\nextra: 1\n"), &r); err == nil {
		t.Fatalf("expected strict-mode error for unknown field")
	}
}

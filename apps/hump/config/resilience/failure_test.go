package resilience

import (
	"testing"

	"hump/pkg/resilience"

	"github.com/goccy/go-yaml"
)

func TestFailureUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want resilience.FailureMode
	}{
		{name: "scalar", src: "skip", want: resilience.FailureModeSkip},
		{name: "mapping", src: "mode: skip\n", want: resilience.FailureModeSkip},
		{name: "mapping default", src: "{}", want: resilience.FailureModeCrash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Failure
			if err := yaml.Unmarshal([]byte(tt.src), &f); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if f.Mode != tt.want {
				t.Fatalf("Mode = %q, want %q", f.Mode, tt.want)
			}
		})
	}
}

func TestFailureUnmarshalRejected(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "scalar non-string", src: "42"},
		{name: "scalar invalid mode", src: "bogus"},
		{name: "mapping unknown field", src: "mode: skip\nextra: 1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Failure
			if err := yaml.Unmarshal([]byte(tt.src), &f); err == nil {
				t.Fatalf("expected error for %q", tt.src)
			}
		})
	}
}

func TestFailureSetDefaultsKeepsMode(t *testing.T) {
	f := Failure{Mode: resilience.FailureModeSkip}
	f.SetDefaults()
	if f.Mode != resilience.FailureModeSkip {
		t.Fatalf("Mode = %q, want %q", f.Mode, resilience.FailureModeSkip)
	}
}

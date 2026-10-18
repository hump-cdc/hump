package funcs

import (
	"reflect"
	"testing"
)

func TestUnavailableIdentifiesUndeliveredValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"unavailable", undelivered{}, true},
		{"delivered", delivered{}, false},
		{"nil", nil, false},
		{"empty string", "", false},
		{"plain struct", struct{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tmplUnavailable(tt.value); got != tt.want {
				t.Errorf("tmplUnavailable(%#v) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}

func TestUnavailableCountsAsEmpty(t *testing.T) {
	if !tmplEmpty(undelivered{}) {
		t.Error("tmplEmpty(undelivered{}) = false, want true")
	}

	if tmplEmpty(delivered{}) {
		t.Error("tmplEmpty(delivered{}) = true, want false")
	}

	if got := tmplCoalesce(undelivered{}, "fallback"); got != "fallback" {
		t.Errorf("tmplCoalesce() = %#v, want the fallback", got)
	}

	if _, err := tmplRequired("blob is required", undelivered{}); err == nil {
		t.Error("tmplRequired() error = nil, want the value rejected as empty")
	}
}

func TestUnavailableComparesAsEmptyString(t *testing.T) {
	tests := []struct {
		name  string
		other string
		want  bool
	}{
		{"empty string", "", true},
		{"non-empty string", "set", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tmplEq(reflectValue(undelivered{}), reflectValue(tt.other))
			if err != nil {
				t.Fatalf("tmplEq() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("tmplEq(unavailable, %q) = %t, want %t", tt.other, got, tt.want)
			}

			got, err = tmplNe(reflectValue(undelivered{}), reflectValue(tt.other))
			if err != nil {
				t.Fatalf("tmplNe() error = %v", err)
			}

			if got == tt.want {
				t.Errorf("tmplNe(unavailable, %q) = %t, want %t", tt.other, got, !tt.want)
			}
		})
	}
}

// undelivered stands in for a value the source could not deliver.
type undelivered struct{}

// IsUnavailable implements unavailableValue.
func (undelivered) IsUnavailable() bool { return true }

// delivered implements unavailableValue but reports a real value, guarding
// against a check that only looks for the method.
type delivered struct{}

// IsUnavailable implements unavailableValue.
func (delivered) IsUnavailable() bool { return false }

// reflectValue wraps v the way text/template hands arguments to eq and ne.
func reflectValue(v any) reflect.Value {
	return reflect.ValueOf(v)
}

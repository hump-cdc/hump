package funcs

import (
	"errors"
	"strings"
	"testing"
)

func TestYAMLTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"yaml compact", `{{ yaml false .Value }}`, V(map[string]any{"a": 1}), "a: 1"},
		{"yaml pretty indent", `{{ yaml true .Value }}`,
			V(map[string]any{"nested": map[string]any{"k": "v"}}),
			"nested:\n    k: v"},
		{"fromyaml index", `{{ index (fromyaml .Value) "a" }}`, V("a: 1\nb: two"), "1"},
		{"fromyamlarray index", `{{ index (fromyamlarray .Value) 1 }}`, V("- 1\n- 2\n- a"), "2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, tc.data)
			if got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

func TestYAMLTemplateErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
	}{
		{"fromyaml malformed", `{{ fromyaml .Value }}`, V(":\n: : :")},
		{"fromyamlarray not array", `{{ fromyamlarray .Value }}`, V("a: not array")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := render(t, tc.src, tc.data); err == nil {
				t.Fatalf("%s: expected error", tc.src)
			}
		})
	}
}

func TestYAMLTemplateMarshalError(t *testing.T) {
	for _, src := range []string{`{{ yaml false .Value }}`, `{{ yaml true .Value }}`} {
		_, err := render(t, src, V(failingYAML{}))
		if err == nil || !strings.Contains(err.Error(), errYAMLMarshal.Error()) {
			t.Errorf("%s error = %v, want %v", src, err, errYAMLMarshal)
		}
	}
}

// errYAMLMarshal is what failingYAML fails to marshal with.
var errYAMLMarshal = errors.New("yaml marshal failed")

// failingYAML is a value whose MarshalYAML always fails.
type failingYAML struct{}

// MarshalYAML returns errYAMLMarshal.
func (failingYAML) MarshalYAML() (any, error) { return nil, errYAMLMarshal }

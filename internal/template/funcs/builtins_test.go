package funcs

import (
	"testing"
)

func TestBuiltinTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"and last when all true", `{{ and .A .B .C }}`, map[string]any{"A": "x", "B": "y", "C": "z"}, "z"},
		{"and short-circuit empty", `{{ and .A .B }}`, map[string]any{"A": "", "B": "y"}, ""},
		{"or first true", `{{ or .A .B .C }}`, map[string]any{"A": "", "B": "y", "C": "z"}, "y"},
		{"or all empty", `{{ or .A .B }}`, map[string]any{"A": "", "B": ""}, ""},
		{"not false → true", `{{ not .Value }}`, V(false), "true"},
		{"not non-empty → false", `{{ not .Value }}`, V("x"), "false"},

		{"eq", `{{ eq 1 1 }}`, nil, "true"},
		{"eq many false", `{{ eq 1 2 }}`, nil, "false"},
		{"eq many true", `{{ eq "a" "b" "a" }}`, nil, "true"},
		{"ne", `{{ ne 1 2 }}`, nil, "true"},
		{"lt", `{{ lt 1 2 }}`, nil, "true"},
		{"le eq", `{{ le 2 2 }}`, nil, "true"},
		{"gt", `{{ gt 3 2 }}`, nil, "true"},
		{"ge eq", `{{ ge 2 2 }}`, nil, "true"},

		{"len slice", `{{ len .Value }}`, V([]int{1, 2, 3}), "3"},
		{"len string", `{{ len "hello" }}`, nil, "5"},

		{"index slice", `{{ index .Value 1 }}`, V([]string{"a", "b", "c"}), "b"},
		{"index nested map",
			`{{ index .Value "a" "b" }}`,
			V(map[string]any{"a": map[string]any{"b": "deep"}}),
			"deep"},

		{"slice subrange", `{{ slice .Value 1 3 }}`, V([]int{1, 2, 3, 4}), "[2 3]"},
		{"slice string", `{{ slice "hello" 1 4 }}`, nil, "ell"},

		{"call", `{{ call .Add 2 3 }}`, map[string]any{
			"Add": func(a, b int) int { return a + b },
		}, "5"},
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

package funcs

import (
	"testing"
)

func TestFlowTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"empty nil", `{{ empty .Value }}`, V(nil), "true"},
		{"empty string", `{{ empty "" }}`, nil, "true"},
		{"empty pipeline string", `{{ .Value | empty }}`, V(""), "true"},
		{"empty zero int", `{{ empty .Value }}`, V(0), "true"},
		{"empty zero uint", `{{ empty .Value }}`, V(uint(0)), "true"},
		{"empty zero float", `{{ empty .Value }}`, V(float64(0)), "true"},
		{"empty false", `{{ empty .Value }}`, V(false), "true"},
		{"empty empty slice", `{{ empty .Value }}`, V([]int{}), "true"},
		{"empty empty map", `{{ empty .Value }}`, V(map[string]int{}), "true"},
		{"non-empty string", `{{ empty "x" }}`, nil, "false"},
		{"non-empty int", `{{ empty .Value }}`, V(7), "false"},
		{"non-empty uint", `{{ empty .Value }}`, V(uint(3)), "false"},
		{"non-empty float", `{{ empty .Value }}`, V(float64(1.5)), "false"},
		{"non-empty slice", `{{ empty .Value }}`, V([]int{1}), "false"},
		{"non-empty map", `{{ empty .Value }}`, V(map[string]int{"a": 1}), "false"},
		{"struct false", `{{ empty .Value }}`, V(struct{ X int }{}), "false"},

		{"required ok", `{{ required "missing" .Value }}`, V("x"), "x"},

		{"coalesce picks first non-empty", `{{ coalesce .A .B .C }}`, map[string]any{"A": "", "B": nil, "C": "yes"}, "yes"},
		{"coalesce all empty", `{{ coalesce "" 0 nil }}`, nil, "<no value>"},

		{"ternary true", `{{ ternary "y" "n" true }}`, nil, "y"},
		{"ternary false", `{{ ternary "y" "n" false }}`, nil, "n"},

		{"default fallback", `{{ default "fb" .Missing }}`, map[string]any{}, "fb"},
		{"default passthrough", `{{ default "fb" .Value }}`, V("x"), "x"},
		{"default empty string passthrough", `{{ default "fb" .Value }}`, V(""), ""},
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

func TestFlowTemplate_Errors(t *testing.T) {
	if _, err := render(t, `{{ required "name is required" .Value }}`, V("")); err == nil {
		t.Fatalf("expected required error")
	}
	if _, err := render(t, `{{ required "missing" .Missing }}`, map[string]any{}); err == nil {
		t.Fatalf("expected required error for nil")
	}
	if _, err := render(t, `{{ fail "boom" }}`, nil); err == nil {
		t.Fatalf("expected fail error")
	}
}

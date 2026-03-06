package funcs

import "testing"

func TestMathTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"add", `{{ add 1 2 3 }}`, nil, "6"},
		{"add empty", `{{ add }}`, nil, "0"},
		{"add1", `{{ add1 .Value }}`, V(5), "6"},
		{"sub", `{{ sub 10 3 }}`, nil, "7"},
		{"mul", `{{ mul 2 3 4 }}`, nil, "24"},
		{"mul empty", `{{ mul }}`, nil, "0"},
		{"div", `{{ div 10 3 }}`, nil, "3"},
		{"div by zero", `{{ div 10 0 }}`, nil, "0"},
		{"mod", `{{ mod 10 3 }}`, nil, "1"},
		{"mod by zero", `{{ mod 10 0 }}`, nil, "0"},
		{"max", `{{ max 1 5 3 }}`, nil, "5"},
		{"max empty", `{{ max }}`, nil, "0"},
		{"min", `{{ min 5 1 3 }}`, nil, "1"},
		{"min empty", `{{ min }}`, nil, "0"},

		{"addf", `{{ addf 1.5 2.5 }}`, nil, "4"},
		{"add1f", `{{ add1f .Value }}`, V(1.5), "2.5"},
		{"subf", `{{ subf 5.5 1.5 }}`, nil, "4"},
		{"mulf", `{{ mulf 2.0 3.0 }}`, nil, "6"},
		{"mulf empty", `{{ mulf }}`, nil, "0"},
		{"divf", `{{ divf 7.0 2.0 }}`, nil, "3.5"},
		{"divf zero", `{{ divf 7.0 0 }}`, nil, "0"},
		{"maxf", `{{ maxf 1.5 2.5 1.0 }}`, nil, "2.5"},
		{"maxf empty", `{{ maxf }}`, nil, "0"},
		{"minf", `{{ minf 2.5 1.5 3.0 }}`, nil, "1.5"},
		{"minf empty", `{{ minf }}`, nil, "0"},

		{"floor", `{{ floor .Value }}`, V(3.9), "3"},
		{"ceil", `{{ ceil .Value }}`, V(3.1), "4"},
		{"round 0", `{{ round .Value 0 }}`, V(3.456), "3"},
		{"round 2", `{{ round .Value 2 }}`, V(3.456), "3.46"},
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

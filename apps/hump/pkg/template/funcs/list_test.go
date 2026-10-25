package funcs

import "testing"

func TestListTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"list literal", `{{ list 1 2 3 }}`, nil, "[1 2 3]"},
		{"first", `{{ first .Value }}`, V([]any{1, 2, 3}), "1"},
		{"first empty", `{{ first .Value }}`, V([]any{}), "<no value>"},
		{"last", `{{ last .Value }}`, V([]any{1, 2, 3}), "3"},
		{"last empty", `{{ last .Value }}`, V([]any{}), "<no value>"},
		{"rest", `{{ rest .Value }}`, V([]any{1, 2, 3}), "[2 3]"},
		{"rest single", `{{ rest .Value }}`, V([]any{1}), "[]"},
		{"initial", `{{ initial .Value }}`, V([]any{1, 2, 3}), "[1 2]"},
		{"initial single", `{{ initial .Value }}`, V([]any{1}), "[]"},
		{"append", `{{ append .Value 4 5 }}`, V([]any{1, 2, 3}), "[1 2 3 4 5]"},
		{"prepend", `{{ prepend .Value 0 }}`, V([]any{1, 2}), "[0 1 2]"},
		{"concat", `{{ concat .Value (list 4 5) }}`, V([]any{1, 2}), "[1 2 4 5]"},
		{"reverse", `{{ reverse .Value }}`, V([]any{1, 2, 3}), "[3 2 1]"},
		{"uniq", `{{ uniq .Value }}`, V([]any{1, 1, 2, 3, 2}), "[1 2 3]"},
		{"uniq non-comparable passes through", `{{ uniq .Value }}`, V([]any{1, []int{1, 2}, 1}), "[1 [1 2]]"},
		{"compact", `{{ compact .Value }}`, V([]any{0, 1, "", "x", nil, false, "y"}), "[1 x y]"},
		{"without", `{{ without .Value 2 }}`, V([]any{1, 2, 3, 2}), "[1 3]"},
		{"has true", `{{ has 2 .Value }}`, V([]any{1, 2, 3}), "true"},
		{"has false", `{{ has 99 .Value }}`, V([]any{1, 2}), "false"},
		{"until", `{{ until 3 }}`, nil, "[0 1 2]"},
		{"until zero", `{{ until 0 }}`, nil, "[]"},
		{"until negative", `{{ until -3 }}`, nil, "[0 -1 -2]"},
		{"untilstep", `{{ untilstep 1 10 2 }}`, nil, "[1 3 5 7 9]"},
		{"untilstep down", `{{ untilstep 10 0 -3 }}`, nil, "[10 7 4 1]"},
		{"untilstep step zero", `{{ untilstep 0 10 0 }}`, nil, "[]"},
		{"seq one", `{{ seq 3 }}`, nil, "[1 2 3]"},
		{"seq two", `{{ seq 2 5 }}`, nil, "[2 3 4 5]"},
		{"seq two desc", `{{ seq 5 2 }}`, nil, "[5 4 3 2]"},
		{"seq three", `{{ seq 1 2 5 }}`, nil, "[1 3 5]"},
		{"seq step zero", `{{ seq 1 0 5 }}`, nil, "[]"},
		{"chunk", `{{ chunk 2 .Value }}`, V([]any{1, 2, 3, 4, 5}), "[[1 2] [3 4] [5]]"},
		{"len", `{{ len .Value }}`, V([]any{"a", "b"}), "2"},
		{"index slice", `{{ index .Value 1 }}`, V([]string{"a", "b", "c"}), "b"},

		// typed (non-[]any) slices/arrays exercise the reflection coercion path.
		{"typed slice", `{{ reverse .Value }}`, V([]int{1, 2, 3}), "[3 2 1]"},
		{"fixed array", `{{ last .Value }}`, V([2]string{"a", "b"}), "b"},
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

func TestListTemplate_Errors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
	}{
		{"chunk size", `{{ chunk 0 .Value }}`, V([]any{1})},
		{"chunk type", `{{ chunk 2 .Value }}`, V("not a list")},
		{"first type", `{{ first .Value }}`, V("not a list")},
		{"last type", `{{ last .Value }}`, V("not a list")},
		{"rest type", `{{ rest .Value }}`, V("not a list")},
		{"initial type", `{{ initial .Value }}`, V("not a list")},
		{"append type", `{{ append .Value 1 }}`, V("not a list")},
		{"prepend type", `{{ prepend .Value 1 }}`, V("not a list")},
		{"concat type", `{{ concat .Value }}`, V("not a list")},
		{"reverse type", `{{ reverse .Value }}`, V("not a list")},
		{"uniq type", `{{ uniq .Value }}`, V("not a list")},
		{"compact type", `{{ compact .Value }}`, V("not a list")},
		{"without type", `{{ without .Value 1 }}`, V("not a list")},
		{"has type", `{{ has 1 .Value }}`, V("not a list")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := render(t, tc.src, tc.data); err == nil {
				t.Fatalf("%s: expected error", tc.src)
			}
		})
	}
}

func TestSeqWithoutArgsAndToAnySliceNil(t *testing.T) {
	if got := mustRender(t, `{{ seq }}`, nil); got != "[]" {
		t.Fatalf("seq = %q, want []", got)
	}
	if xs, err := toAnySlice(nil); xs != nil || err != nil {
		t.Fatalf("toAnySlice(nil) = %v, %v, want nil, nil", xs, err)
	}
}

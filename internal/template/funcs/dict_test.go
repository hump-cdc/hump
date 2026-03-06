package funcs

import "testing"

func TestDictTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{
			name: "dict + get",
			src:  `{{ get (dict "a" 1 "b" "two") "b" }}`,
			data: nil,
			want: "two",
		},
		{
			name: "dict non-string key coerced",
			src:  `{{ get (dict 7 "v") "7" }}`,
			data: nil,
			want: "v",
		},
		{
			name: "set then get",
			src:  `{{ get (set .Value "k" "v") "k" }}`,
			data: V(map[string]any{}),
			want: "v",
		},
		{
			name: "unset then haskey",
			src:  `{{ haskey (unset .Value "a") "a" }}`,
			data: V(map[string]any{"a": 1, "b": 2}),
			want: "false",
		},
		{
			name: "haskey true",
			src:  `{{ haskey .Value "a" }}`,
			data: V(map[string]any{"a": 1}),
			want: "true",
		},
		{
			name: "keys sorted",
			src:  `{{ keys .Value }}`,
			data: V(map[string]any{"b": 2, "a": 1, "c": 3}),
			want: "[a b c]",
		},
		{
			name: "values key-sorted",
			src:  `{{ values .Value }}`,
			data: V(map[string]any{"b": 2, "a": 1, "c": 3}),
			want: "[1 2 3]",
		},
		{
			name: "pick subset",
			src:  `{{ keys (pick .Value "a" "c" "z") }}`,
			data: V(map[string]any{"a": 1, "b": 2, "c": 3}),
			want: "[a c]",
		},
		{
			name: "pluck",
			src:  `{{ pluck "a" (dict "a" 1) (dict "a" 2 "b" 9) (dict "b" 3) }}`,
			data: nil,
			want: "[1 2]",
		},
		{
			name: "dig hit",
			src:  `{{ dig "a" "b" "c" "fb" .Value }}`,
			data: V(map[string]any{"a": map[string]any{"b": map[string]any{"c": "v"}}}),
			want: "v",
		},
		{
			name: "dig fallback",
			src:  `{{ dig "a" "x" "y" "fb" .Value }}`,
			data: V(map[string]any{"a": map[string]any{"b": "ok"}}),
			want: "fb",
		},
		{
			name: "dig non-string key coerced",
			src:  `{{ dig 7 "k" "fb" .Value }}`,
			data: V(map[string]any{"7": map[string]any{"k": "v"}}),
			want: "v",
		},
		{
			name: "dig non-map descent returns fallback",
			src:  `{{ dig "a" "b" "fb" .Value }}`,
			data: V(map[string]any{"a": "leaf"}),
			want: "fb",
		},
		{
			name: "merge keeps existing",
			src:  `{{ get (merge (dict "a" 1) (dict "a" 99 "b" 2)) "a" }}`,
			data: nil,
			want: "1",
		},
		{
			name: "merge adds new key",
			src:  `{{ get (merge (dict "a" 1) (dict "b" 2)) "b" }}`,
			data: nil,
			want: "2",
		},
		{
			name: "mergeoverwrite replaces",
			src:  `{{ get (mergeoverwrite (dict "a" 1) (dict "a" 99)) "a" }}`,
			data: nil,
			want: "99",
		},
		{
			name: "omit case-insensitive",
			src:  `{{ keys (omit .Value "password" "USER") }}`,
			data: V(map[string]any{"Password": "x", "User": "y", "keep": 1}),
			want: "[keep]",
		},
		{
			name: "omit non-map passthrough",
			src:  `{{ omit .Value "x" }}`,
			data: V("scalar"),
			want: "scalar",
		},
		{
			name: "get pipe form",
			src:  `{{ .Value | get "b" }}`,
			data: V(map[string]any{"a": 1, "b": "two"}),
			want: "two",
		},
		{
			name: "set pipe form",
			src:  `{{ get (.Value | set "k" "v") "k" }}`,
			data: V(map[string]any{}),
			want: "v",
		},
		{
			name: "unset pipe form",
			src:  `{{ haskey (.Value | unset "a") "a" }}`,
			data: V(map[string]any{"a": 1, "b": 2}),
			want: "false",
		},
		{
			name: "haskey pipe form",
			src:  `{{ .Value | haskey "a" }}`,
			data: V(map[string]any{"a": 1}),
			want: "true",
		},
		{
			name: "pick pipe form",
			src:  `{{ keys (.Value | pick "a" "c" "z") }}`,
			data: V(map[string]any{"a": 1, "b": 2, "c": 3}),
			want: "[a c]",
		},
		{
			name: "omit pipe form",
			src:  `{{ keys (.Value | omit "version") }}`,
			data: V(map[string]any{"id": 1, "version": 7, "name": "x"}),
			want: "[id name]",
		},
		{
			name: "deepcopy map",
			src:  `{{ get (deepcopy .Value) "a" }}`,
			data: V(map[string]any{"a": 1}),
			want: "1",
		},
		{
			name: "deepcopy nested slice value",
			src:  `{{ index (get (deepcopy .Value) "list") 0 }}`,
			data: V(map[string]any{"list": []any{1, map[string]any{"k": "v"}}}),
			want: "1",
		},
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

func TestDictTemplate_Errors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
	}{
		{"dict odd args", `{{ dict "a" }}`, nil},
		{"dig too few args", `{{ dig "x" .Value }}`, V(map[string]any{"x": 1})},
		{"dig non-map", `{{ dig "a" "fb" .Value }}`, V("not-a-map")},
		{"get no map", `{{ get "nomap" }}`, nil},
		{"get bad arity", `{{ get .Value "a" "b" }}`, V(map[string]any{"a": 1})},
		{"set no map", `{{ set "nomap" }}`, nil},
		{"set bad arity", `{{ set .Value "a" }}`, V(map[string]any{"a": 1})},
		{"unset no map", `{{ unset "nomap" }}`, nil},
		{"unset bad arity", `{{ unset .Value "a" "b" }}`, V(map[string]any{"a": 1})},
		{"haskey no map", `{{ haskey "nomap" }}`, nil},
		{"haskey bad arity", `{{ haskey .Value "a" "b" }}`, V(map[string]any{"a": 1})},
		{"pick no map", `{{ pick "nomap" }}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := render(t, tc.src, tc.data); err == nil {
				t.Fatalf("%s: expected error", tc.src)
			}
		})
	}
}

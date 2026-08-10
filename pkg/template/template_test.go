package template

import (
	"hump/pkg/template/funcs"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestIsTemplate(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		expect bool
	}{
		// Negatives: no "{{" → short-circuit to false.
		{name: "plain/empty", src: "", expect: false},
		{name: "plain/plain text", src: "hello world", expect: false},
		{name: "plain/single brace", src: "{ .foo }", expect: false},
		{name: "plain/mustache-ish but no double-brace", src: "{ {.foo} }", expect: false},
		{name: "plain/only closing braces", src: "no open }} here", expect: false},
		{name: "plain/sql-looking literal", src: "SELECT * FROM t WHERE id = 1", expect: false},

		// Positives: contains "{{" and parses.
		{name: "template/dot cursor", src: "{{.}}", expect: true},
		{name: "template/field access", src: "{{ .table }}", expect: true},
		{name: "template/nested field", src: "{{ .values.id }}", expect: true},
		{name: "template/range end", src: "{{ range .values }}{{ . }}{{ end }}", expect: true},
		{name: "template/with end", src: "{{ with .values }}{{ .x }}{{ end }}", expect: true},
		{name: "template/if else end", src: "{{ if .x }}a{{ else }}b{{ end }}", expect: true},
		{name: "template/pipeline", src: `{{ .name | printf "%q" }}`, expect: true},
		{name: "template/variable assign", src: "{{ $x := .table }}{{ $x }}", expect: true},
		{name: "template/trim markers", src: "x {{- .table -}} y", expect: true},
		{name: "template/comment", src: "{{/* note */}}", expect: true},
		{name: "template/text around action", src: "events.{{.table}}.v1", expect: true},
		{name: "template/escaped open brace literal", src: `{{"{{"}}`, expect: true},
		{name: "template/unknown root still parses", src: "{{ .whatever }}", expect: true},

		// Broken templates: "{{" present but parse fails → false.
		{name: "plain/unterminated action", src: "{{ .values.id ", expect: false},
		{name: "plain/empty action", src: "{{}}", expect: false},
		{name: "plain/bad keyword", src: "{{ range }}{{ end }}", expect: false},
		{name: "plain/mismatched end", src: "{{ if .x }}no-end", expect: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, got := isTemplate(tc.src); got != tc.expect {
				t.Fatalf("isTemplate(%q) = %v, expect %v", tc.src, got, tc.expect)
			}
		})
	}
}

func TestNew_Static(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{name: "empty", src: ""},
		{name: "plain text", src: "hello world"},
		{name: "sql literal", src: "SELECT * FROM t WHERE id = 1"},
		{name: "single brace", src: "{ .foo }"},
		{name: "broken action falls back to static", src: "{{ .values.id "},
		{name: "empty action falls back to static", src: "{{}}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := New(tc.src)
			if err != nil {
				t.Fatalf("New(%q) unexpected error: %v", tc.src, err)
			}
			if _, ok := tmpl.tmpl.(*templateStatic); !ok {
				t.Fatalf("New(%q) = %T, want *templateStatic", tc.src, tmpl.tmpl)
			}
			if tmpl.IsDynamic() {
				t.Fatalf("IsDynamic() = true for static src %q", tc.src)
			}
			if got := tmpl.Source(); got != tc.src {
				t.Fatalf("Source() = %q, want %q", got, tc.src)
			}
			got, err := tmpl.Render(nil)
			if err != nil {
				t.Fatalf("Render unexpected error: %v", err)
			}
			if got != tc.src {
				t.Fatalf("Render = %q, want %q", got, tc.src)
			}
		})
	}
}

func TestNew_Dynamic(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{
			name: "field access",
			src:  "{{ .Table }}",
			data: map[string]any{"Table": "users"},
			want: "users",
		},
		{
			name: "text around action",
			src:  "events.{{.Table}}.v1",
			data: map[string]any{"Table": "orders"},
			want: "events.orders.v1",
		},
		{
			name: "range",
			src:  "{{range .Items}}{{.}},{{end}}",
			data: map[string]any{"Items": []string{"a", "b", "c"}},
			want: "a,b,c,",
		},
		{
			name: "if else",
			src:  "{{if .Flag}}yes{{else}}no{{end}}",
			data: map[string]any{"Flag": false},
			want: "no",
		},
		{
			name: "pipeline printf",
			src:  `{{ .Name | printf "%q" }}`,
			data: map[string]any{"Name": "hump"},
			want: `"hump"`,
		},
		{
			name: "nil data with literal action",
			src:  `{{"{{"}}`,
			data: nil,
			want: "{{",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := New(tc.src)
			if err != nil {
				t.Fatalf("New(%q) unexpected error: %v", tc.src, err)
			}
			if _, ok := tmpl.tmpl.(templateRenderable); !ok {
				t.Fatalf("New(%q) = %T, want templateRenderable", tc.src, tmpl.tmpl)
			}
			if !tmpl.IsDynamic() {
				t.Fatalf("IsDynamic() = false for dynamic src %q", tc.src)
			}
			if got := tmpl.Source(); got != tc.src {
				t.Fatalf("Source() = %q, want %q", got, tc.src)
			}
			got, err := tmpl.Render(tc.data)
			if err != nil {
				t.Fatalf("Render unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Render = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRender_DynamicError(t *testing.T) {
	// Execute-time error: index out of range on a non-empty slice.
	tmpl, err := New(`{{ index .V 10 }}`)
	if err != nil {
		t.Fatalf("New unexpected error: %v", err)
	}
	if _, err := tmpl.Render(map[string]any{"V": []int{1, 2}}); err == nil {
		t.Fatalf("expected Render error, got nil")
	}
}

func TestNilReceiver(t *testing.T) {
	var t1 *Template
	if got, err := t1.Render(map[string]any{"x": 1}); err != nil || got != "" {
		t.Fatalf("nil.Render() = (%q, %v), want (\"\", nil)", got, err)
	}
	if got := t1.Source(); got != "" {
		t.Fatalf("nil.Source() = %q, want \"\"", got)
	}
	if t1.IsDynamic() {
		t.Fatalf("nil.IsDynamic() = true, want false")
	}
}

func TestMarshalYAML(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{name: "static", src: "hello"},
		{name: "dynamic", src: "events.{{ .Table }}.v1"},
		{name: "empty", src: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := New(tc.src)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			v, err := tmpl.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML: %v", err)
			}
			s, ok := v.(string)
			if !ok {
				t.Fatalf("MarshalYAML returned %T, want string", v)
			}
			if s != tc.src {
				t.Fatalf("MarshalYAML = %q, want %q", s, tc.src)
			}
		})
	}
}

func TestMarshalYAML_RoundTrip(t *testing.T) {
	type holder struct {
		Tmpl Template `yaml:"tmpl"`
	}

	src := "events.{{ .Table }}.v1"
	in := holder{}
	tmpl, err := New(src)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	in.Tmpl = *tmpl

	out, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	// Marshaled string contains the source. Don't pin exact YAML format
	// (quoting/escaping), just verify the source value is present.
	if !strings.Contains(string(out), "{{ .Table }}") {
		t.Fatalf("marshaled yaml missing source: %q", out)
	}

	var got holder
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if got.Tmpl.Source() != src {
		t.Fatalf("round-trip Source() = %q, want %q", got.Tmpl.Source(), src)
	}
	if !got.Tmpl.IsDynamic() {
		t.Fatalf("round-trip IsDynamic() = false, want true")
	}
	rendered, err := got.Tmpl.Render(map[string]any{"Table": "users"})
	if err != nil {
		t.Fatalf("Render after round-trip: %v", err)
	}
	if rendered != "events.users.v1" {
		t.Fatalf("Render = %q, want %q", rendered, "events.users.v1")
	}
}

func TestUnmarshalYAML_Static(t *testing.T) {
	type holder struct {
		Tmpl Template `yaml:"tmpl"`
	}

	var h holder
	if err := yaml.Unmarshal([]byte("tmpl: hello world\n"), &h); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if h.Tmpl.Source() != "hello world" {
		t.Fatalf("Source() = %q, want %q", h.Tmpl.Source(), "hello world")
	}
	if h.Tmpl.IsDynamic() {
		t.Fatalf("IsDynamic() = true for static source")
	}
}

func TestUnmarshalYAML_NonScalar(t *testing.T) {
	type holder struct {
		Tmpl Template `yaml:"tmpl"`
	}

	// A mapping node where a scalar is expected must error.
	src := "tmpl:\n  a: 1\n"
	var h holder
	err := yaml.Unmarshal([]byte(src), &h)
	if err == nil {
		t.Fatalf("expected error decoding non-scalar into Template, got nil")
	}
	if !strings.Contains(err.Error(), "should be string") {
		t.Fatalf("error %q does not mention 'should be string'", err.Error())
	}
}

func TestUnmarshalYAML_NonStringScalar(t *testing.T) {
	type holder struct {
		Tmpl Template `yaml:"tmpl"`
	}

	// A non-string scalar (integer) reaches UnmarshalYAML as a scalar node
	// whose GetValue() is not a string, so it must error.
	var h holder
	err := yaml.Unmarshal([]byte("tmpl: 123\n"), &h)
	if err == nil {
		t.Fatalf("expected error decoding integer scalar into Template, got nil")
	}
	if !strings.Contains(err.Error(), "should be string") {
		t.Fatalf("error %q does not mention 'should be string'", err.Error())
	}
}

func TestUnmarshalYAML_BrokenTemplateFallsBackToStatic(t *testing.T) {
	type holder struct {
		Tmpl Template `yaml:"tmpl"`
	}

	// "{{ .x " contains "{{" but does not parse — isTemplate returns false,
	// so it falls back to a static template containing the raw source.
	src := "tmpl: \"{{ .x \"\n"
	var h holder
	if err := yaml.Unmarshal([]byte(src), &h); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if h.Tmpl.IsDynamic() {
		t.Fatalf("IsDynamic() = true, want false")
	}
	if h.Tmpl.Source() != "{{ .x " {
		t.Fatalf("Source() = %q, want %q", h.Tmpl.Source(), "{{ .x ")
	}
}

// TestNew_FuncMap_AllCallable invokes every name registered in funcs.Funcs
// through New().Render to confirm each helper is wired up and callable. Edge
// cases are owned by the per-package tests in funcs/.
func TestNew_FuncMap_AllCallable(t *testing.T) {
	t.Setenv("HUMP_TEMPLATE_TEST", "envvalue")

	deterministic := []struct {
		fn   string
		src  string
		data any
		want string
	}{
		// stdlib builtins
		{"and", `{{ and "a" "b" }}`, nil, "b"},
		{"call", `{{ call .F 2 3 }}`, map[string]any{"F": func(a, b int) int { return a + b }}, "5"},
		{"eq", `{{ eq 1 1 }}`, nil, "true"},
		{"ge", `{{ ge 2 2 }}`, nil, "true"},
		{"gt", `{{ gt 3 2 }}`, nil, "true"},
		{"html", `{{ html "<a>" }}`, nil, "&lt;a&gt;"},
		{"index", `{{ index .V 1 }}`, map[string]any{"V": []int{10, 20, 30}}, "20"},
		{"js", `{{ js "a'b" }}`, nil, `a\'b`},
		{"le", `{{ le 2 2 }}`, nil, "true"},
		{"len", `{{ len "hello" }}`, nil, "5"},
		{"lt", `{{ lt 1 2 }}`, nil, "true"},
		{"ne", `{{ ne 1 2 }}`, nil, "true"},
		{"not", `{{ not false }}`, nil, "true"},
		{"or", `{{ or "" "x" }}`, nil, "x"},
		{"print", `{{ print "a" "b" }}`, nil, "ab"},
		{"printf", `{{ printf "%d-%s" 1 "x" }}`, nil, "1-x"},
		{"println", `{{ println "x" }}`, nil, "x\n"},
		{"slice", `{{ slice "hello" 1 4 }}`, nil, "ell"},
		{"urlquery", `{{ urlquery "a b" }}`, nil, "a+b"},

		// flow
		{"coalesce", `{{ coalesce "" 0 "first" }}`, nil, "first"},
		{"default", `{{ default "fb" .Missing }}`, map[string]any{}, "fb"},
		{"empty", `{{ empty "" }}`, nil, "true"},
		{"required", `{{ required "must" "ok" }}`, nil, "ok"},
		{"ternary", `{{ ternary "y" "n" true }}`, nil, "y"},

		// case (direct call form)
		{"camelcase", `{{ camelcase "foo bar" }}`, nil, "fooBar"},
		{"capitalcase", `{{ capitalcase "foo bar" }}`, nil, "Foo Bar"},
		{"constantcase", `{{ constantcase "foo bar" }}`, nil, "FOO_BAR"},
		{"dotcase", `{{ dotcase "foo bar" }}`, nil, "foo.bar"},
		{"headercase", `{{ headercase "foo bar" }}`, nil, "Foo-Bar"},
		{"kebabcase", `{{ kebabcase "foo bar" }}`, nil, "foo-bar"},
		{"lower", `{{ lower "FooBar" }}`, nil, "foo bar"},
		{"lowerfirst", `{{ lowerfirst "Hello" }}`, nil, "hello"},
		{"pascalcase", `{{ pascalcase "foo bar" }}`, nil, "FooBar"},
		{"pascalsnakecase", `{{ pascalsnakecase "foo bar" }}`, nil, "Foo_Bar"},
		{"pathcase", `{{ pathcase "foo bar" }}`, nil, "foo/bar"},
		{"snakecase", `{{ snakecase "fooBar" }}`, nil, "foo_bar"},
		{"upper", `{{ upper "fooBar" }}`, nil, "FOO BAR"},
		{"upperfirst", `{{ upperfirst "hello" }}`, nil, "Hello"},

		// strings
		{"abbrev", `{{ abbrev 8 "abcdefghij" }}`, nil, "abcde..."},
		{"abbrevboth", `{{ abbrevboth 5 10 "abcdefghijklmnop" }}`, nil, "...fghi..."},
		{"cat", `{{ cat "a" 1 true }}`, nil, "a 1 true"},
		{"contains", `{{ contains "ell" "hello" }}`, nil, "true"},
		{"hasprefix", `{{ hasprefix "foo" "foobar" }}`, nil, "true"},
		{"hassuffix", `{{ hassuffix "bar" "foobar" }}`, nil, "true"},
		{"indent", `{{ indent 2 "a\nb" }}`, nil, "  a\n  b"},
		{"initials", `{{ initials "John Ronald Reuel" }}`, nil, "JRR"},
		{"nindent", `{{ nindent 2 "a" }}`, nil, "\n  a"},
		{"nospace", `{{ nospace "a b\tc" }}`, nil, "abc"},
		{"plural", `{{ plural "x" "xs" 2 }}`, nil, "xs"},
		{"quote", `{{ quote "a" }}`, nil, `"a"`},
		{"repeat", `{{ repeat 3 "ab" }}`, nil, "ababab"},
		{"replace", `{{ replace "a" "X" "banana" }}`, nil, "bXnXnX"},
		{"squote", `{{ squote "a" }}`, nil, `'a'`},
		{"substr", `{{ substr 1 4 "hello" }}`, nil, "ell"},
		{"swapcase", `{{ swapcase "Ab" }}`, nil, "aB"},
		{"trim", `{{ trim "  x  " }}`, nil, "x"},
		{"trimall", `{{ trimall "xy" "xyhelloyx" }}`, nil, "hello"},
		{"trimprefix", `{{ trimprefix "foo" "foobar" }}`, nil, "bar"},
		{"trimsuffix", `{{ trimsuffix "bar" "foobar" }}`, nil, "foo"},
		{"trunc", `{{ trunc 3 "abcdef" }}`, nil, "abc"},
		{"wrap", `{{ wrap 5 "hello world" }}`, nil, "hello\nworld"},
		{"wrapwith", `{{ wrapwith 5 " | " "hello world" }}`, nil, "hello | world"},

		// convert
		{"atoi", `{{ atoi "42" }}`, nil, "42"},
		{"float64", `{{ float64 "2.5" }}`, nil, "2.5"},
		{"fromjson", `{{ index (fromjson "{\"a\":1}") "a" }}`, nil, "1"},
		{"fromjsonarray", `{{ index (fromjsonarray "[1,2,3]") 1 }}`, nil, "2"},
		{"int", `{{ int "12" }}`, nil, "12"},
		{"int64", `{{ int64 "12" }}`, nil, "12"},
		{"json", `{{ json false (dict "a" 1) }}`, nil, `{"a":1}`},
		{"todecimal", `{{ todecimal "777" }}`, nil, "511"},
		{"tostring", `{{ tostring 42 }}`, nil, "42"},
		{"tostrings", `{{ tostrings (list 1 "x") }}`, nil, "[1 x]"},

		// yaml
		{"yaml", `{{ yaml false (dict "a" 1) }}`, nil, "a: 1"},
		{"fromyaml", `{{ index (fromyaml "a: 1") "a" }}`, nil, "1"},
		{"fromyamlarray", `{{ index (fromyamlarray "- 1\n- 2") 1 }}`, nil, "2"},

		// regex
		{"regexfind", `{{ regexfind "\\d+" "a 12 b" }}`, nil, "12"},
		{"regexfindall", `{{ regexfindall "\\d+" -1 "1 2 3" }}`, nil, "[1 2 3]"},
		{"regexmatch", `{{ regexmatch "\\d+" "abc 1" }}`, nil, "true"},
		{"regexreplaceall", `{{ regexreplaceall "\\d" "X" "a1b2" }}`, nil, "aXbX"},
		{"regexreplaceallliteral", `{{ regexreplaceallliteral "\\d" "$1" "a1" }}`, nil, "a$1"},
		{"regexsplit", `{{ regexsplit "," -1 "a,b,c" }}`, nil, "[a b c]"},

		// encoding
		{"b64enc", `{{ b64enc "hello" }}`, nil, "aGVsbG8="},
		{"b64dec", `{{ b64dec "aGVsbG8=" }}`, nil, "hello"},
		{"b32enc", `{{ b32enc "" }}`, nil, ""},
		{"b32dec", `{{ b32dec (b32enc "x") }}`, nil, "x"},

		// crypto deterministic
		{"sha1", `{{ sha1 "abc" }}`, nil, "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"sha256", `{{ sha256 "abc" }}`, nil, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"adler32", `{{ adler32 "abc" }}`, nil, "38600999"},

		// time deterministic
		{"date", `{{ date "2006-01-02" .V }}`, map[string]any{"V": "2024-01-02T00:00:00Z"}, "2024-01-02"},
		{"dateinzone", `{{ dateinzone "2006-01-02" .V "UTC" }}`, map[string]any{"V": "2024-01-02T00:00:00Z"}, "2024-01-02"},
		{"datemodify", `{{ (datemodify "1h" .V).Hour }}`, map[string]any{"V": "2024-01-02T00:00:00Z"}, "1"},
		{"duration", `{{ duration 90 }}`, nil, "1m30s"},
		{"htmldate", `{{ htmldate .V }}`, map[string]any{"V": "2024-01-02T00:00:00Z"}, "2024-01-02"},
		{"htmldateinzone", `{{ htmldateinzone .V "UTC" }}`, map[string]any{"V": "2024-01-02T00:00:00Z"}, "2024-01-02"},
		{"todate", `{{ (todate "2006-01-02" "2024-01-02").Year }}`, nil, "2024"},
		{"unixepoch", `{{ unixepoch .V }}`, map[string]any{"V": "2024-01-02T00:00:00Z"}, "1704153600"},

		// dict
		{"dict", `{{ index (dict "a" 1) "a" }}`, nil, "1"},
		{"get", `{{ get (dict "a" 1) "a" }}`, nil, "1"},
		{"haskey", `{{ haskey (dict "a" 1) "a" }}`, nil, "true"},
		{"set", `{{ get (set (dict) "k" "v") "k" }}`, nil, "v"},
		{"unset", `{{ haskey (unset (dict "a" 1) "a") "a" }}`, nil, "false"},
		{"keys", `{{ keys (dict "b" 2 "a" 1) }}`, nil, "[a b]"},
		{"values", `{{ values (dict "b" 2 "a" 1) }}`, nil, "[1 2]"},
		{"pick", `{{ keys (pick (dict "a" 1 "b" 2) "a") }}`, nil, "[a]"},
		{"pluck", `{{ pluck "a" (dict "a" 1) (dict "a" 2) }}`, nil, "[1 2]"},
		{"dig", `{{ dig "a" "fb" (dict "a" "v") }}`, nil, "v"},
		{"merge", `{{ get (merge (dict "a" 1) (dict "b" 2)) "b" }}`, nil, "2"},
		{"mergeoverwrite", `{{ get (mergeoverwrite (dict "a" 1) (dict "a" 9)) "a" }}`, nil, "9"},
		{"omit", `{{ keys (omit (dict "a" 1 "b" 2) "a") }}`, nil, "[b]"},
		{"deepcopy", `{{ get (deepcopy (dict "a" 1)) "a" }}`, nil, "1"},

		// list
		{"list", `{{ list 1 2 3 }}`, nil, "[1 2 3]"},
		{"first", `{{ first (list 1 2 3) }}`, nil, "1"},
		{"last", `{{ last (list 1 2 3) }}`, nil, "3"},
		{"rest", `{{ rest (list 1 2 3) }}`, nil, "[2 3]"},
		{"initial", `{{ initial (list 1 2 3) }}`, nil, "[1 2]"},
		{"append", `{{ append (list 1) 2 }}`, nil, "[1 2]"},
		{"prepend", `{{ prepend (list 2) 1 }}`, nil, "[1 2]"},
		{"concat", `{{ concat (list 1) (list 2) }}`, nil, "[1 2]"},
		{"reverse", `{{ reverse (list 1 2 3) }}`, nil, "[3 2 1]"},
		{"uniq", `{{ uniq (list 1 1 2) }}`, nil, "[1 2]"},
		{"compact", `{{ compact (list 0 1 "" "x") }}`, nil, "[1 x]"},
		{"without", `{{ without (list 1 2 3) 2 }}`, nil, "[1 3]"},
		{"has", `{{ has 2 (list 1 2 3) }}`, nil, "true"},
		{"chunk", `{{ chunk 2 (list 1 2 3) }}`, nil, "[[1 2] [3]]"},
		{"until", `{{ until 3 }}`, nil, "[0 1 2]"},
		{"untilstep", `{{ untilstep 0 6 2 }}`, nil, "[0 2 4]"},
		{"seq", `{{ seq 3 }}`, nil, "[1 2 3]"},

		// int math
		{"add", `{{ add 1 2 3 }}`, nil, "6"},
		{"add1", `{{ add1 4 }}`, nil, "5"},
		{"sub", `{{ sub 10 3 }}`, nil, "7"},
		{"mul", `{{ mul 2 3 }}`, nil, "6"},
		{"div", `{{ div 10 3 }}`, nil, "3"},
		{"mod", `{{ mod 10 3 }}`, nil, "1"},
		{"max", `{{ max 1 5 3 }}`, nil, "5"},
		{"min", `{{ min 5 1 3 }}`, nil, "1"},

		// float math
		{"addf", `{{ addf 1.5 2.5 }}`, nil, "4"},
		{"add1f", `{{ add1f 1.5 }}`, nil, "2.5"},
		{"subf", `{{ subf 5.5 1.5 }}`, nil, "4"},
		{"mulf", `{{ mulf 2.0 3.0 }}`, nil, "6"},
		{"divf", `{{ divf 7.0 2.0 }}`, nil, "3.5"},
		{"maxf", `{{ maxf 1.5 2.5 }}`, nil, "2.5"},
		{"minf", `{{ minf 2.5 1.5 }}`, nil, "1.5"},
		{"floor", `{{ floor 3.9 }}`, nil, "3"},
		{"ceil", `{{ ceil 3.1 }}`, nil, "4"},
		{"round", `{{ round 3.456 2 }}`, nil, "3.46"},

		// filepath
		{"base", `{{ base "/a/b.txt" }}`, nil, "b.txt"},
		{"clean", `{{ clean "/a/./b" }}`, nil, "/a/b"},
		{"dir", `{{ dir "/a/b.txt" }}`, nil, "/a"},
		{"ext", `{{ ext "a.txt" }}`, nil, ".txt"},
		{"isabs", `{{ isabs "/x" }}`, nil, "true"},

		// url
		{"urlparse", `{{ (urlparse "https://x/p").host }}`, nil, "x"},
		{"urljoin", `{{ urljoin (dict "scheme" "https" "host" "x") }}`, nil, "https://x"},

		// env
		{"env", `{{ env "HUMP_TEMPLATE_TEST" }}`, nil, "envvalue"},

		// tracing
		{"trace_id", `{{ trace_id "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" }}`, nil, "4bf92f3577b34da6a3ce929d0e0e4736"},
		{"span_id", `{{ span_id "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" }}`, nil, "00f067aa0ba902b7"},
	}

	for _, tc := range deterministic {
		t.Run(tc.fn, func(t *testing.T) {
			tmpl, err := New(tc.src)
			if err != nil {
				t.Fatalf("New(%q): %v", tc.src, err)
			}
			got, err := tmpl.Render(tc.data)
			if err != nil {
				t.Fatalf("Render(%q): %v", tc.src, err)
			}
			if got != tc.want {
				t.Fatalf("%s: %q = %q, want %q", tc.fn, tc.src, got, tc.want)
			}
		})
	}

	// Non-deterministic helpers: assert the call succeeds and produces output
	// that matches a shape constraint, without pinning exact values.
	nondet := []struct {
		fn    string
		src   string
		check func(t *testing.T, out string)
	}{
		{
			fn:  "now",
			src: `{{ (now).Year }}`,
			check: func(t *testing.T, out string) {
				if out == "" {
					t.Fatalf("now produced empty year")
				}
			},
		},
		{
			fn:  "ago",
			src: `{{ ago (now) }}`,
			check: func(t *testing.T, out string) {
				if out == "" {
					t.Fatalf("ago produced empty")
				}
			},
		},
		{
			fn:  "durationround",
			src: `{{ durationround (now) }}`,
			check: func(t *testing.T, out string) {
				if out == "" {
					t.Fatalf("durationround produced empty")
				}
			},
		},
		{
			fn:  "uuidv4",
			src: `{{ uuidv4 }}`,
			check: func(t *testing.T, out string) {
				re := regexp.MustCompile(`^[0-9a-f-]{36}$`)
				if !re.MatchString(out) {
					t.Fatalf("uuidv4 = %q", out)
				}
			},
		},
		{
			fn:  "randalpha",
			src: `{{ randalpha 8 }}`,
			check: func(t *testing.T, out string) {
				if len(out) != 8 {
					t.Fatalf("randalpha len = %d", len(out))
				}
			},
		},
		{
			fn:  "randalphanum",
			src: `{{ randalphanum 8 }}`,
			check: func(t *testing.T, out string) {
				if len(out) != 8 {
					t.Fatalf("randalphanum len = %d", len(out))
				}
			},
		},
		{
			fn:  "randascii",
			src: `{{ randascii 8 }}`,
			check: func(t *testing.T, out string) {
				if len(out) != 8 {
					t.Fatalf("randascii len = %d", len(out))
				}
			},
		},
		{
			fn:  "randnumeric",
			src: `{{ randnumeric 8 }}`,
			check: func(t *testing.T, out string) {
				if len(out) != 8 {
					t.Fatalf("randnumeric len = %d", len(out))
				}
			},
		},
		{
			fn:  "randbytes",
			src: `{{ randbytes 16 }}`,
			check: func(t *testing.T, out string) {
				if out == "" {
					t.Fatalf("randbytes empty")
				}
			},
		},
		{
			fn:  "shuffle",
			src: `{{ shuffle "abcdef" }}`,
			check: func(t *testing.T, out string) {
				if len(out) != 6 {
					t.Fatalf("shuffle len = %d", len(out))
				}
			},
		},
		{
			fn:  "hostname",
			src: `{{ hostname }}`,
			check: func(t *testing.T, out string) {
				if out == "" {
					t.Fatalf("hostname empty")
				}
			},
		},
		{
			fn:  "aes_encrypt/aes_decrypt",
			src: `{{ aes_decrypt "k" (aes_encrypt "k" "secret-msg") }}`,
			check: func(t *testing.T, out string) {
				if out != "secret-msg" {
					t.Fatalf("aes round-trip = %q", out)
				}
			},
		},
	}
	for _, tc := range nondet {
		t.Run(tc.fn, func(t *testing.T) {
			tmpl, err := New(tc.src)
			if err != nil {
				t.Fatalf("New(%q): %v", tc.src, err)
			}
			got, err := tmpl.Render(nil)
			if err != nil {
				t.Fatalf("Render(%q): %v", tc.src, err)
			}
			tc.check(t, got)
		})
	}

	// fail must propagate as a render error.
	t.Run("fail", func(t *testing.T) {
		tmpl, err := New(`{{ fail "boom" }}`)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, err := tmpl.Render(nil); err == nil {
			t.Fatalf("fail must error")
		}
	})

	// Sanity: every name in funcs.Funcs is covered by either the deterministic
	// or nondet table (or by the explicit "fail" subtest).
	covered := map[string]struct{}{"fail": {}}
	for _, c := range deterministic {
		covered[c.fn] = struct{}{}
	}
	for _, c := range nondet {
		covered[c.fn] = struct{}{}
	}
	// nondet "aes_encrypt/aes_decrypt" covers two names.
	covered["aes_encrypt"] = struct{}{}
	covered["aes_decrypt"] = struct{}{}
	for name := range funcs.Funcs {
		if _, ok := covered[name]; !ok {
			t.Errorf("function %q is registered but not invoked by this test", name)
		}
	}
}

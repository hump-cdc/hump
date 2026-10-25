package funcs

import (
	"os"
	"strings"
	"testing"
	"text/template"
)

// TestFuncs_AllRegistered ensures every advertised template function name is
// actually present in the FuncMap.
func TestFuncs_AllRegistered(t *testing.T) {
	expected := []string{
		// stdlib builtins
		"and", "call", "eq", "ge", "gt", "html", "index", "js", "le", "len",
		"lt", "ne", "not", "or", "print", "printf", "println", "slice", "urlquery",
		// flow
		"coalesce", "default", "empty", "fail", "required", "ternary",
		// case
		"camelcase", "capitalcase", "constantcase", "dotcase", "headercase",
		"kebabcase", "lower", "lowerfirst", "pascalcase", "pascalsnakecase",
		"pathcase", "snakecase", "upper", "upperfirst",
		// strings
		"abbrev", "abbrevboth", "cat", "contains", "hasprefix", "hassuffix",
		"indent", "initials", "nindent", "nospace", "plural", "quote", "repeat",
		"replace", "shuffle", "squote", "substr", "swapcase", "trim", "trimall",
		"trimprefix", "trimsuffix", "trunc", "wrap", "wrapwith",
		"randalpha", "randalphanum", "randascii", "randnumeric",
		// convert
		"atoi", "float64", "fromjson", "fromjsonarray", "int", "int64", "json",
		"todecimal", "tostring", "tostrings",
		// yaml
		"fromyaml", "fromyamlarray", "yaml",
		// regex
		"regexfind", "regexfindall", "regexmatch", "regexreplaceall",
		"regexreplaceallliteral", "regexsplit",
		// encoding
		"b32dec", "b32enc", "b64dec", "b64enc",
		// crypto
		"adler32", "aes_decrypt", "aes_encrypt", "randbytes", "sha256",
		// time
		"ago", "date", "dateinzone", "datemodify", "duration", "durationround",
		"htmldate", "htmldateinzone", "now", "todate", "unixepoch",
		// dict
		"deepcopy", "dict", "dig", "get", "haskey", "keys", "merge",
		"mergeoverwrite", "omit", "pick", "pluck", "set", "unset", "values",
		// list
		"append", "chunk", "compact", "concat", "first", "has", "initial",
		"last", "list", "prepend", "rest", "reverse", "seq", "uniq", "until",
		"untilstep", "without",
		// math (int)
		"add", "add1", "div", "max", "min", "mod", "mul", "sub",
		// math (float)
		"add1f", "addf", "ceil", "divf", "floor", "maxf", "minf", "mulf", "round", "subf",
		// filepath
		"base", "clean", "dir", "ext", "isabs",
		// url
		"urljoin", "urlparse",
		// uuid
		"uuidv4",
		// os/env
		"env", "hostname",
	}
	for _, name := range expected {
		if _, ok := Funcs[name]; !ok {
			t.Errorf("missing func: %q", name)
		}
	}
}

// TestFuncs_RealTemplate exercises a representative subset of helpers through
// the actual text/template engine to catch signature regressions.
func TestFuncs_RealTemplate(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`{{ snakecase "fooBar" }}`, "foo_bar"},
		{`{{ "hello" | upper }}`, "HELLO"},
		{`{{ b64enc "hello" }}`, "aGVsbG8="},
		{`{{ sha256 "abc" }}`, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{`{{ list 1 2 3 | len }}`, "3"},
		{`{{ add 1 2 3 }}`, "6"},
		{`{{ default "fb" "" }}`, ""}, // empty string passes through
		{`{{ default "fb" .Missing }}`, "fb"},
		{`{{ ternary "y" "n" true }}`, "y"},
		{`{{ index (list 10 20 30) 1 }}`, "20"},
		{`{{ "abc" | quote }}`, `"abc"`},
		{`{{ trim "  hi  " }}`, "hi"},
		{`{{ uniq (list 1 1 2 3 2) }}`, "[1 2 3]"},
		{`{{ first (list "a" "b") }}`, "a"},
		{`{{ regexmatch "^[a-z]+$" "abc" }}`, "true"},
		{`{{ contains "ell" "hello" }}`, "true"},
		{`{{ ext "a/b.txt" }}`, ".txt"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			if got := mustRender(t, tc.src, map[string]any{}); got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

// TestFuncs_Hostname verifies the "hostname" helper resolves to os.Hostname().
func TestFuncs_Hostname(t *testing.T) {
	want, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname: %v", err)
	}
	if got := mustRender(t, `{{ hostname }}`, nil); got != want {
		t.Fatalf("hostname = %q, want %q", got, want)
	}
}

func FuzzFuncs_RowValue(f *testing.F) {
	f.Add("hello world", 3)
	f.Add("FooBar_baz-qux", 7)
	f.Add("", 0)
	f.Add("Ünïcødé ДАННЫЕ 数据", 12)
	f.Add("\xff\xfe\x00 invalid utf-8", -4)
	f.Add("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", 55)
	f.Add("../../etc/passwd", 1000)

	templates := make([]*template.Template, len(rowValueTemplates))
	for i, src := range rowValueTemplates {
		templates[i] = template.Must(template.New(src).Funcs(Funcs).Parse(src))
	}

	f.Fuzz(func(t *testing.T, value string, n int) {
		data := map[string]any{"Value": value, "N": n % 1000}

		for _, tmpl := range templates {
			var got, again strings.Builder
			if err := tmpl.Execute(&got, data); err != nil {
				t.Fatalf("render(%q) with %q error = %v", tmpl.Name(), value, err)
			}

			if err := tmpl.Execute(&again, data); err != nil || again.String() != got.String() {
				t.Fatalf("render(%q) with %q = %q, then %q, %v, want it deterministic", tmpl.Name(), value, got.String(), again.String(), err)
			}
		}
	})
}

// rowValueTemplates apply every function that must accept any string row
// value, as a template author would write them, without failing.
var rowValueTemplates = []string{
	`{{ .Value | camelcase }}`, `{{ .Value | capitalcase }}`, `{{ .Value | constantcase }}`,
	`{{ .Value | dotcase }}`, `{{ .Value | headercase }}`, `{{ .Value | kebabcase }}`,
	`{{ .Value | lower }}`, `{{ .Value | lowerfirst }}`, `{{ .Value | pascalcase }}`,
	`{{ .Value | pascalsnakecase }}`, `{{ .Value | pathcase }}`, `{{ .Value | snakecase }}`,
	`{{ .Value | upper }}`, `{{ .Value | upperfirst }}`,
	`{{ .Value | initials }}`, `{{ .Value | nospace }}`, `{{ .Value | quote }}`, `{{ .Value | squote }}`,
	`{{ .Value | swapcase }}`, `{{ .Value | trim }}`, `{{ .Value | trimall "x" }}`,
	`{{ .Value | trimprefix "x-" }}`, `{{ .Value | trimsuffix "-x" }}`, `{{ .Value | replace "a" "b" }}`,
	`{{ .Value | hasprefix "x" }}`, `{{ .Value | hassuffix "x" }}`, `{{ .Value | contains "x" }}`,
	`{{ .Value | abbrev .N }}`, `{{ .Value | abbrevboth 4 .N }}`, `{{ .Value | trunc .N }}`,
	`{{ .Value | substr 0 .N }}`, `{{ .Value | substr .N -1 }}`, `{{ .Value | wrap .N }}`,
	`{{ .Value | wrapwith .N "|" }}`, `{{ .Value | indent 2 }}`, `{{ .Value | nindent 2 }}`,
	`{{ .Value | repeat 2 }}`, `{{ cat .Value .Value }}`, `{{ plural "one" "many" .N }}`,
	`{{ .Value | b64enc }}`, `{{ .Value | b64dec }}`, `{{ .Value | b32enc }}`, `{{ .Value | b32dec }}`,
	`{{ .Value | sha256 }}`, `{{ .Value | adler32 }}`,
	`{{ .Value | base }}`, `{{ .Value | clean }}`, `{{ .Value | dir }}`, `{{ .Value | ext }}`, `{{ .Value | isabs }}`,
	`{{ .Value | trace_id }}`, `{{ .Value | span_id }}`,
	`{{ .Value | json }}`, `{{ .Value | yaml false }}`, `{{ .Value | tostring }}`,
	`{{ .Value | atoi }}`, `{{ .Value | int }}`, `{{ .Value | int64 }}`, `{{ .Value | float64 }}`, `{{ .Value | todecimal }}`,
	`{{ .Value | html }}`, `{{ .Value | js }}`, `{{ .Value | urlquery }}`, `{{ len .Value }}`,
	`{{ .Value | regexmatch "^[a-z]+$" }}`, `{{ .Value | regexreplaceall "[0-9]+" "#" }}`,
	`{{ .Value | regexfindall "[a-z]+" -1 }}`, `{{ .Value | regexsplit "-" -1 }}`,
	`{{ eq .Value "x" }}`, `{{ .Value | empty }}`, `{{ coalesce .Value "fallback" }}`,
}

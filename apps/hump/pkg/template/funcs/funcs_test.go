package funcs

import (
	"os"
	"testing"
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

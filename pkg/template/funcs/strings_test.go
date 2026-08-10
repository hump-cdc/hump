package funcs

import (
	"slices"
	"strings"
	"testing"
)

func TestStringsTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"trim", `{{ .Value | trim }}`, "  hello  ", "hello"},
		{"trimall", `{{ trimall "xy" .Value }}`, "xyhelloyx", "hello"},
		{"trimprefix", `{{ trimprefix "foo" .Value }}`, "foobar", "bar"},
		{"trimprefix none", `{{ trimprefix "zzz" .Value }}`, "foobar", "foobar"},
		{"trimsuffix", `{{ trimsuffix "bar" .Value }}`, "foobar", "foo"},
		{"swapcase", `{{ .Value | swapcase }}`, "HelloWorld 123", "hELLOwORLD 123"},
		{"substr", `{{ substr 1 4 .Value }}`, "hello", "ell"},
		{"substr neg end", `{{ substr 1 -1 .Value }}`, "hello", "ell"},
		{"substr neg start", `{{ substr -2 4 .Value }}`, "hello", "hell"},
		{"substr clamp end", `{{ substr 0 100 .Value }}`, "hello", "hello"},
		{"substr inverted", `{{ substr 4 1 .Value }}`, "hello", ""},
		{"substr unicode", `{{ substr 1 4 .Value }}`, "héllo", "éll"},
		{"trunc pos", `{{ trunc 3 .Value }}`, "abcdef", "abc"},
		{"trunc len passthrough", `{{ trunc 10 .Value }}`, "abc", "abc"},
		{"trunc neg", `{{ trunc -3 .Value }}`, "abcdef", "def"},
		{"trunc neg over", `{{ trunc -99 .Value }}`, "abc", "abc"},
		{"abbrev", `{{ abbrev 8 .Value }}`, "abcdefghij", "abcde..."},
		{"abbrev short width", `{{ abbrev 3 .Value }}`, "abcdefghij", "abcdefghij"},
		{"abbrev short input", `{{ abbrev 20 .Value }}`, "short", "short"},
		{"abbrevboth offset bad", `{{ abbrevboth 3 7 .Value }}`, "abcdefghij", "abcdefghij"},
		{"abbrevboth maxw bad", `{{ abbrevboth 5 6 .Value }}`, "abcdefghij", "abcdefghij"},
		{"abbrevboth short", `{{ abbrevboth 4 7 .Value }}`, "abcdef", "abcdef"},
		{"abbrevboth left anchored", `{{ abbrevboth 4 7 .Value }}`, "abcdefghij", "abcd..."},
		{"abbrevboth clamped both sides", `{{ abbrevboth 8 7 .Value }}`, "abcdefghij", "...g..."},
		{"abbrevboth both sides", `{{ abbrevboth 5 10 .Value }}`, "1234567890123456789012345678901234567890", "...6789..."},
		{"nospace", `{{ .Value | nospace }}`, "a b\tc\nd", "abcd"},
		{"initials", `{{ .Value | initials }}`, "John Ronald Reuel", "JRR"},
		{"initials empty", `{{ .Value | initials }}`, "", ""},
		{"wrap", `{{ wrap 5 .Value }}`, "hello world foo bar", "hello\nworld\nfoo\nbar"},
		{"wrap nonpositive width", `{{ wrap -1 .Value }}`, "hello", "hello"},
		{"wrap empty", `{{ wrap 10 .Value }}`, "", ""},
		{"wrapwith", `{{ wrapwith 5 " | " .Value }}`, "hello world", "hello | world"},
		{"wrapwith packing", `{{ wrapwith 11 "/" .Value }}`, "hi yo there", "hi yo there"},
		{"wrapwith long word", `{{ wrapwith 3 "/" .Value }}`, "aaaa bb c", "aaaa/bb/c"},
		{"wrapwith multiline", `{{ wrapwith 5 "\n" .Value }}`, "hello world\n\nfoo", "hello\nworld\n\nfoo"},
		{"contains true", `{{ contains "foo" .Value }}`, "foobar", "true"},
		{"hasprefix", `{{ hasprefix "foo" .Value }}`, "foobar", "true"},
		{"hassuffix", `{{ hassuffix "bar" .Value }}`, "foobar", "true"},
		{"quote", `{{ quote "a" "b" 1 }}`, nil, `"a" "b" "1"`},
		{"squote", `{{ squote "a" 1 }}`, nil, `'a' '1'`},
		{"cat", `{{ cat "a" 1 true }}`, nil, "a 1 true"},
		{"indent", `{{ indent 2 .Value }}`, "a\nb\nc", "  a\n  b\n  c"},
		{"nindent", `{{ nindent 2 .Value }}`, "a\nb", "\n  a\n  b"},
		{"indent zero", `{{ indent 0 .Value }}`, "x", "x"},
		{"indent nil value", `{{ indent 2 .Value }}`, nil, "  "},
		{"repeat", `{{ repeat 3 .Value }}`, "ab", "ababab"},
		{"repeat zero", `{{ repeat 0 .Value }}`, "ab", ""},
		{"repeat neg", `{{ repeat -1 .Value }}`, "ab", ""},
		{"repeat nil", `{{ repeat 2 .Value }}`, nil, ""},
		{"replace", `{{ replace "a" "X" .Value }}`, "banana", "bXnXnX"},
		{"plural one", `{{ plural "apple" "apples" 1 }}`, nil, "apple"},
		{"plural many", `{{ plural "apple" "apples" 2 }}`, nil, "apples"},
		{"plural zero", `{{ plural "apple" "apples" 0 }}`, nil, "apples"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, V(tc.in))
			if got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

func TestStringsRandTemplates(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		alphabetSet string
		wantLen     int
	}{
		{"randalpha", `{{ randalpha .Value }}`, asciiLetters, 10},
		{"randalphanum", `{{ randalphanum .Value }}`, asciiLetters + asciiDigits, 10},
		{"randnumeric", `{{ randnumeric .Value }}`, asciiDigits, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, V(tc.wantLen))
			if len(got) != tc.wantLen {
				t.Fatalf("%s len = %d, want %d", tc.src, len(got), tc.wantLen)
			}
			for _, r := range got {
				if !strings.ContainsRune(tc.alphabetSet, r) {
					t.Fatalf("%s: rune %q not in alphabet", tc.src, r)
				}
			}
		})
	}
	if got := mustRender(t, `{{ randascii .Value }}`, V(20)); len(got) != 20 {
		t.Fatalf("randascii len = %d", len(got))
	} else {
		for _, r := range got {
			if r < 32 || r > 126 {
				t.Fatalf("randascii non-printable: %d", r)
			}
		}
	}
	if got := mustRender(t, `{{ randalpha .Value }}`, V(0)); got != "" {
		t.Fatalf("zero len want empty")
	}
	if got := mustRender(t, `{{ randascii .Value }}`, V(-1)); got != "" {
		t.Fatalf("neg len want empty")
	}
}

func TestShuffleTemplate(t *testing.T) {
	in := "abcdefghijkl"
	got := mustRender(t, `{{ shuffle .Value }}`, V(in))
	if len(got) != len(in) {
		t.Fatalf("len changed: %q", got)
	}
	gotBytes := []byte(got)
	inBytes := []byte(in)
	slices.Sort(gotBytes)
	slices.Sort(inBytes)
	if string(gotBytes) != string(inBytes) {
		t.Fatalf("rune set differs: %q vs %q", got, in)
	}
}

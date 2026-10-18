package funcs

import "testing"

func TestRegexTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"match true", `{{ regexmatch "\\d+" .Value }}`, "abc 123", "true"},
		{"match false", `{{ regexmatch "\\d+" .Value }}`, "abc", "false"},
		{"find", `{{ regexfind "\\d+" .Value }}`, "a 12 b 34", "12"},
		{"find none", `{{ regexfind "\\d+" .Value }}`, "no digits", ""},
		{"findAll all", `{{ regexfindall "\\d+" -1 .Value }}`, "a 12 b 34 c 56", "[12 34 56]"},
		{"findAll capped", `{{ regexfindall "\\d+" 2 .Value }}`, "1 2 3 4", "[1 2]"},
		{"replaceAll", `{{ regexreplaceall "(\\w+)@(\\w+)" "$2/$1" .Value }}`, "x@y z@w", "y/x w/z"},
		{"replaceAllLiteral", `{{ regexreplaceallliteral "\\d+" "$1" .Value }}`, "a 12 b", "a $1 b"},
		{"split all", `{{ regexsplit "," -1 .Value }}`, "a,b,c,d", "[a b c d]"},
		{"split capped", `{{ index (regexsplit "," 2 .Value) 1 }}`, "a,b,c,d", "b,c,d"},
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

func TestRegexTemplate_Errors(t *testing.T) {
	bad := []string{
		`{{ regexmatch "(" .Value }}`,
		`{{ regexfind "(" .Value }}`,
		`{{ regexfindall "(" 1 .Value }}`,
		`{{ regexreplaceall "(" "" .Value }}`,
		`{{ regexreplaceallliteral "(" "" .Value }}`,
		`{{ regexsplit "(" -1 .Value }}`,
	}
	for _, src := range bad {
		if _, err := render(t, src, V("x")); err == nil {
			t.Fatalf("%s: expected compile error", src)
		}
	}
}

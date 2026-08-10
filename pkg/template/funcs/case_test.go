package funcs

import "testing"

func TestCaseTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"camelcase", `{{ .Value | camelcase }}`, "get url path", "getUrlPath"},
		{"camelcase from pascal", `{{ .Value | camelcase }}`, "GetURLPath", "getUrlPath"},
		{"camelcase nil", `{{ .Value | camelcase }}`, nil, ""},
		{"pascalcase", `{{ .Value | pascalcase }}`, "get url path", "GetUrlPath"},
		{"snakecase", `{{ .Value | snakecase }}`, "GetURLPath", "get_url_path"},
		{"constantcase", `{{ .Value | constantcase }}`, "fooBar", "FOO_BAR"},
		{"kebabcase", `{{ .Value | kebabcase }}`, "fooBar", "foo-bar"},
		{"upper", `{{ .Value | upper }}`, "fooBar", "FOO BAR"},
		{"lower", `{{ .Value | lower }}`, "FooBar", "foo bar"},
		{"capitalcase", `{{ .Value | capitalcase }}`, "foo bar", "Foo Bar"},
		{"dotcase", `{{ .Value | dotcase }}`, "fooBar", "foo.bar"},
		{"headercase", `{{ .Value | headercase }}`, "fooBar", "Foo-Bar"},
		{"pathcase", `{{ .Value | pathcase }}`, "fooBar", "foo/bar"},
		{"pascalsnakecase", `{{ .Value | pascalsnakecase }}`, "fooBar", "Foo_Bar"},
		{"lowerfirst", `{{ .Value | lowerfirst }}`, "Hello", "hello"},
		{"lowerfirst empty", `{{ .Value | lowerfirst }}`, "", ""},
		{"upperfirst", `{{ .Value | upperfirst }}`, "hello", "Hello"},
		{"upperfirst empty", `{{ .Value | upperfirst }}`, "", ""},

		// Word tokenization exercised through snakecase rendering.
		{"split empty", `{{ .Value | snakecase }}`, "", ""},
		{"split acronym to title boundary", `{{ .Value | snakecase }}`, "XMLParser", "xml_parser"},
		{"split leading acronym then word", `{{ .Value | snakecase }}`, "HTTPServer", "http_server"},
		{"split digit letter boundary", `{{ .Value | snakecase }}`, "foo2bar", "foo_2_bar"},
		{"split lower upper boundary", `{{ .Value | snakecase }}`, "fooBar", "foo_bar"},
		{"split punctuation separators", `{{ .Value | snakecase }}`, "foo--bar..baz", "foo_bar_baz"},
		{"split leading punctuation", `{{ .Value | snakecase }}`, "--foo", "foo"},
		{"split trailing punctuation", `{{ .Value | snakecase }}`, "foo--", "foo"},
		{"split all punctuation", `{{ .Value | snakecase }}`, "----", ""},
		{"split mixed acronym digits", `{{ .Value | snakecase }}`, "parseURL2HTML", "parse_url_2_html"},

		// Title-casing of words exercised through capitalcase rendering.
		{"title single rune", `{{ .Value | capitalcase }}`, "a", "A"},
		{"title acronym", `{{ .Value | capitalcase }}`, "URL", "Url"},
		{"title word", `{{ .Value | capitalcase }}`, "foo", "Foo"},
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

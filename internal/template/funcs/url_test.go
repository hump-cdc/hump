package funcs

import (
	"strings"
	"testing"
)

func TestURLTemplates_Parse(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"scheme", `{{ (urlparse .Value).scheme }}`, "https://example.com:8080/p?x=1#f", "https"},
		{"host", `{{ (urlparse .Value).host }}`, "https://example.com:8080/p", "example.com:8080"},
		{"hostname", `{{ (urlparse .Value).hostname }}`, "https://example.com:8080/p", "example.com"},
		{"path", `{{ (urlparse .Value).path }}`, "https://example.com/p/q", "/p/q"},
		{"query", `{{ (urlparse .Value).query }}`, "https://x/p?a=1", "a=1"},
		{"query multi", `{{ (urlparse .Value).query }}`, "https://x/p?a=1&b=2", "a=1&b=2"},
		{"fragment", `{{ (urlparse .Value).fragment }}`, "https://x/p#frag", "frag"},
		{"userinfo", `{{ (urlparse .Value).userinfo }}`, "https://u:p@x/", "u:p"},
		{"userinfo empty", `{{ (urlparse .Value).userinfo }}`, "/relative", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, V(tc.in))
			if got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
	if _, err := render(t, `{{ urlparse .Value }}`, V("://bad")); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestURLTemplates_Join(t *testing.T) {
	got := mustRender(t,
		`{{ urljoin (dict "scheme" "https" "host" "x" "path" "/p") }}`, nil)
	if got != "https://x/p" {
		t.Fatalf("got %q", got)
	}
	got = mustRender(t,
		`{{ urljoin (dict "scheme" "https" "host" "x" "userinfo" "u:p") }}`, nil)
	if !strings.HasPrefix(got, "https://u:p@x") {
		t.Fatalf("got %q", got)
	}
}

func TestURLTemplates_JoinUserinfoError(t *testing.T) {
	// An invalid percent-escape in userinfo makes url.Parse fail, exercising
	// the error branch in urljoin.
	if _, err := render(t,
		`{{ urljoin (dict "scheme" "https" "host" "x" "userinfo" "a%zzb") }}`, nil); err == nil {
		t.Fatalf("expected userinfo parse error")
	}
}

func TestURLTemplates_RoundTrip(t *testing.T) {
	in := "https://example.com/a/b?x=1"
	got := mustRender(t, `{{ urljoin (urlparse .Value) }}`, V(in))
	if got != in {
		t.Fatalf("got %q want %q", got, in)
	}
}

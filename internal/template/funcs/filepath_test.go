package funcs

import "testing"

func TestFilepathTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"base", `{{ .Value | base }}`, "/a/b/c.txt", "c.txt"},
		{"dir", `{{ .Value | dir }}`, "/a/b/c.txt", "/a/b"},
		{"clean", `{{ .Value | clean }}`, "/a/./b/../c", "/a/c"},
		{"ext", `{{ .Value | ext }}`, "foo.tar.gz", ".gz"},
		{"ext none", `{{ .Value | ext }}`, "foo", ""},
		{"isabs true", `{{ .Value | isabs }}`, "/x", "true"},
		{"isabs false", `{{ .Value | isabs }}`, "rel/path", "false"},
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

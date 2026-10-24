package funcs

import "testing"

func TestEncodingTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"b64enc", `{{ .Value | b64enc }}`, "hello", "aGVsbG8="},
		{"b64dec", `{{ .Value | b64dec }}`, "aGVsbG8=", "hello"},
		{"b64dec bad", `{{ .Value | b64dec }}`, "@@bad@@", ""},
		{"b64enc empty", `{{ .Value | b64enc }}`, "", ""},
		{"b32 round trip", `{{ b32enc .Value | b32dec }}`, "round-trip", "round-trip"},
		{"b32dec bad", `{{ .Value | b32dec }}`, "!!", ""},
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

func FuzzEncodingRoundTrip(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("\x00\xff\xfe binary")
	f.Add("Ünïcødé")

	f.Fuzz(func(t *testing.T, value string) {
		for _, src := range []string{`{{ .Value | b64enc | b64dec }}`, `{{ .Value | b32enc | b32dec }}`} {
			if got := mustRender(t, src, V(value)); got != value {
				t.Fatalf("%s = %q, want %q", src, got, value)
			}
		}
	})
}

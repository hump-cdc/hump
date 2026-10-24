package funcs

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"text/template"
)

// render parses src with the package FuncMap and executes it against data.
// Returns the rendered string or any parse/execute error.
func render(t *testing.T, src string, data any) (string, error) {
	t.Helper()
	tmpl, err := template.New("").Funcs(Funcs).Parse(src)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// mustRender renders or fails the test. For asserting positive cases.
func mustRender(t *testing.T, src string, data any) string {
	t.Helper()
	out, err := render(t, src, data)
	if err != nil {
		t.Fatalf("render(%q): %v", src, err)
	}
	return out
}

// V wraps a single value as `.Value` for pipeline-style template tests.
func V(v any) map[string]any { return map[string]any{"Value": v} }

// errRandFailed is what the reader withFailingRand installs fails with.
var errRandFailed = errors.New("rand failed")

// withFailingRand replaces crypto/rand.Reader with one that always fails, for
// the duration of the test.
func withFailingRand(t *testing.T) {
	t.Helper()
	orig := rand.Reader
	rand.Reader = failingReader{}
	t.Cleanup(func() { rand.Reader = orig })
}

// failingReader is an io.Reader that always returns errRandFailed.
type failingReader struct{}

// Read returns errRandFailed.
func (failingReader) Read([]byte) (int, error) { return 0, errRandFailed }

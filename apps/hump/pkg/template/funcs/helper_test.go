package funcs

import (
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

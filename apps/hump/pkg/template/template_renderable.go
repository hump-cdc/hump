package template

import (
	"fmt"
	"strings"
	"text/template"
)

// templateRenderable is the dynamic tmpl variant, backed by a parsed
// text/template that is executed against data on each Render call.
type templateRenderable struct {
	tmpl   *template.Template
	source string
}

// Render executes the parsed template against data and returns the result.
// Execution failures are wrapped with [ErrRender].
func (t templateRenderable) Render(data any) (string, error) {
	var sb strings.Builder

	if err := t.tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("%w: %w", ErrRender, err)
	}

	return sb.String(), nil
}

// Source returns the original template source string.
func (t templateRenderable) Source() string { return t.source }

// IsDynamic always returns true for templateRenderable.
func (t templateRenderable) IsDynamic() bool { return true }

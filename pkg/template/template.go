package template

import (
	"fmt"
	"hump/pkg/template/funcs"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml/ast"
)

type tmpl interface {
	Render(data any) (string, error)
	Source() string
	IsDynamic() bool
}

// Template wraps a text/template source that may be either a static string
// or a dynamic Go text/template, chosen automatically at parse time.
type Template struct {
	tmpl
}

// New parses source and returns a Template. If source contains a valid
// "{{ ... }}" action, the result is dynamic; otherwise it is static.
func New(source string) (*Template, error) {
	return parseTemplate(source), nil
}

// Render executes the template with data and returns the result. A nil
// receiver returns an empty string without error.
func (t *Template) Render(data any) (string, error) {
	if t == nil {
		return "", nil
	}

	return t.tmpl.Render(data)
}

// Source returns the original template source string. A nil receiver
// returns an empty string.
func (t *Template) Source() string {
	if t == nil {
		return ""
	}

	return t.tmpl.Source()
}

// IsDynamic reports whether the template contains template actions and
// requires rendering with data. A nil receiver returns false.
func (t *Template) IsDynamic() bool {
	if t == nil {
		return false
	}

	return t.tmpl.IsDynamic()
}

// MarshalYAML implements yaml.Marshaler by emitting the template's source.
func (t Template) MarshalYAML() (any, error) {
	return t.tmpl.Source(), nil
}

// UnmarshalYAML implements yaml.NodeUnmarshaler. It expects a scalar node and
// parses its value as a template source.
func (t *Template) UnmarshalYAML(node ast.Node) error {
	scalar, ok := node.(ast.ScalarNode)
	if !ok {
		return fmt.Errorf("should be string")
	}

	value, ok := scalar.GetValue().(string)
	if !ok {
		return fmt.Errorf("should be string")
	}

	*t = *parseTemplate(value)

	return nil
}

// parseTemplate builds a Template from source, selecting a dynamic
// implementation when isTemplate returns a parsed template and a static one
// otherwise. A malformed action falls back to static, so it never fails.
func parseTemplate(source string) *Template {
	if parsed, ok := isTemplate(source); ok {
		return &Template{tmpl: templateRenderable{tmpl: parsed, source: source}}
	}

	return &Template{tmpl: &templateStatic{source: source}}
}

// isTemplate reports whether source looks like a Go text/template (contains
// "{{") and parses successfully, returning the parsed template when it does.
// It is a cheap dispatcher, not a validator.
func isTemplate(source string) (*template.Template, bool) {
	if !strings.Contains(source, "{{") {
		return nil, false
	}

	tmpl, err := template.New("").Funcs(funcs.Funcs).Parse(source)
	if err != nil {
		return nil, false
	}

	return tmpl, true
}

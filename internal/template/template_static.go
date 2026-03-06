package template

// templateStatic is the non-templated tmpl variant used when the source
// contains no template actions; Render returns the source verbatim.
type templateStatic struct {
	source string
}

// Render returns the source string unchanged; data is ignored.
func (t templateStatic) Render(data any) (string, error) {
	return t.source, nil
}

// Source returns the original source string.
func (t templateStatic) Source() string { return t.source }

// IsDynamic always returns false for templateStatic.
func (t templateStatic) IsDynamic() bool { return false }

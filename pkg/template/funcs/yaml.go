package funcs

import (
	"strings"

	"github.com/goccy/go-yaml"
)

// tmplYAML marshals v to YAML. When prettify is true, the output is indented
// with four-space blocks.
func tmplYAML(prettify bool, v any) (string, error) {
	if !prettify {
		b, err := yaml.Marshal(v)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(b), "\n"), nil
	}
	var sb strings.Builder
	enc := yaml.NewEncoder(&sb, yaml.Indent(4))
	if err := enc.Encode(v); err != nil {
		_ = enc.Close()
		return "", err
	}
	// Close flushes buffered output into the strings.Builder, whose Write
	// never errors; after a successful Encode this cannot fail.
	_ = enc.Close()
	return strings.TrimRight(sb.String(), "\n"), nil
}

// tmplFromYAML decodes a YAML scalar/document string into a generic value.
func tmplFromYAML(s string) (any, error) {
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

// tmplFromYAMLArray decodes a YAML array string into []any.
func tmplFromYAMLArray(s string) ([]any, error) {
	var v []any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

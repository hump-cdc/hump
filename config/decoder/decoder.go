// Package decoder provides the shared YAML decoding entry point used by every
// configuration type: a single validator instance and a strict-mode decoder
// that reports validation failures as token-anchored yaml.Error values.
package decoder

import (
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// Validate is the shared struct validator wired into every decoder so
// `validate:"..."` tags are enforced during unmarshalling.
var Validate = validator.New(validator.WithRequiredStructEnabled())

// Decode decodes node into out with strict mode and [Validate] wired into the
// decoder, so validation failures surface as token-anchored yaml.Error values.
func Decode(node ast.Node, out any) error {
	return yaml.NodeToValue(node, out, yaml.Strict(), yaml.Validator(Validate))
}

// KeyValue returns the value node mapped to key in node, or nil if key is
// absent.
func KeyValue(node *ast.MappingNode, key string) ast.Node {
	for _, kv := range node.Values {
		if kv.Key.String() == key {
			return kv.Value
		}
	}

	return nil
}

// NodeLine returns the source line of an AST node, or 0 if unavailable.
func NodeLine(node ast.Node) int {
	if node == nil {
		return 0
	}
	tok := node.GetToken()
	if tok == nil || tok.Position == nil {
		return 0
	}
	return tok.Position.Line
}

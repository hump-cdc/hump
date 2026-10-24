// Package decoder provides the shared YAML decoding entry point used by every
// configuration type: a single validator instance and a strict-mode decoder
// that reports validation failures as token-anchored yaml.Error values.
package decoder

import (
	"errors"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"hump/internal/domain"
)

// Validate is the shared struct validator wired into every decoder so
// `validate:"..."` tags are enforced during unmarshalling.
var Validate = validator.New(validator.WithRequiredStructEnabled())

// Decode decodes node into out with strict mode and [Validate] wired into the
// decoder, so validation failures surface as token-anchored yaml.Error values.
// String fields keep a scalar's literal text, and an unknown field is the first
// one in document order.
func Decode(node ast.Node, out any) error {
	err := yaml.NodeToValue(node, out, options()...)

	return firstUnknownField(err, node, reflect.TypeOf(out))
}

// Unmarshal decodes the YAML document in source into out the way [Decode]
// decodes a node.
func Unmarshal(source []byte, out any) error {
	err := yaml.UnmarshalWithOptions(source, out, options()...)

	var unknown *yaml.UnknownFieldError
	if !errors.As(err, &unknown) {
		return err
	}

	file, parseErr := parser.ParseBytes(source, 0)
	if parseErr != nil || len(file.Docs) == 0 {
		return err
	}

	return firstUnknownField(err, file.Docs[0].Body, reflect.TypeOf(out))
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

// options are the decode options every configuration type is decoded with.
func options() []yaml.DecodeOption {
	return []yaml.DecodeOption{
		yaml.Strict(),
		yaml.Validator(Validate),
		yaml.CustomUnmarshaler(decodeScalarText[string]),
		yaml.CustomUnmarshaler(decodeScalarText[domain.TableSchema]),
		yaml.CustomUnmarshaler(decodeScalarText[domain.TableName]),
	}
}

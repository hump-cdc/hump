package decoder

import (
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// decodeScalarText decodes a scalar into a string type as the text it is
// written as, so an unquoted `table: 00000` stays "00000" instead of the
// number goccy would format back. Quoted and block scalars keep their decoded
// value, null leaves out unchanged, and anything else decodes as goccy would.
func decodeScalarText[T ~string](out *T, data []byte) error {
	file, err := parser.ParseBytes(data, 0)
	if err != nil {
		return err
	}

	var node ast.Node
	if len(file.Docs) > 0 {
		node = file.Docs[0].Body
	}

	text, ok := ScalarText(node)
	switch {
	case ok:
		*out = T(text)
	case node == nil || node.Type() == ast.NullType:
	default:
		var s string
		if err := yaml.Unmarshal(data, &s); err != nil {
			return err
		}
		*out = T(s)
	}

	return nil
}

// ScalarText returns the value of a non-null scalar node: the decoded value of
// a string or block scalar, the source text of any other one.
func ScalarText(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.AnchorNode:
		return ScalarText(n.Value)
	case *ast.TagNode:
		return ScalarText(n.Value)
	case *ast.StringNode:
		return n.Value, true
	case *ast.LiteralNode:
		return n.Value.Value, true
	case *ast.NullNode:
		return "", false
	case ast.ScalarNode:
		return n.GetToken().Value, true
	default:
		return "", false
	}
}

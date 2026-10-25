package decoder

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

// firstUnknownField replaces an unknown-field err with one naming the first
// unknown key of root in document order, when that key comes no later than the
// one err names: goccy picks one of a mapping's unknown keys at random, and
// reports a nested one before those of the mapping around it. Any other error
// is returned as is.
func firstUnknownField(err error, root ast.Node, out reflect.Type) error {
	var unknown *yaml.UnknownFieldError
	if !errors.As(err, &unknown) {
		return err
	}

	key := firstUnknownKey(root, out)
	if key == nil || isAfter(key.GetToken(), unknown.Token) {
		return err
	}

	return &yaml.UnknownFieldError{Message: fmt.Sprintf(`unknown field "%s"`, keyText(key)), Token: key.GetToken()}
}

// firstUnknownKey returns the first key, in document order, of a mapping under
// node that the struct it decodes into has no field for. It does not descend
// into types that decode themselves: their own Decode reports their keys.
func firstUnknownKey(node ast.Node, typ reflect.Type) ast.MapKeyNode {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch n := node.(type) {
	case *ast.AnchorNode:
		return firstUnknownKey(n.Value, typ)
	case *ast.TagNode:
		return firstUnknownKey(n.Value, typ)
	case *ast.SequenceNode:
		if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
			return nil
		}

		for _, item := range n.Values {
			if key := firstUnknownKeyIn(item, typ.Elem()); key != nil {
				return key
			}
		}
	case *ast.MappingNode:
		return firstUnknownKeyInMapping(n, typ)
	default:
	}

	return nil
}

// firstUnknownKeyInMapping is firstUnknownKey for a mapping node.
func firstUnknownKeyInMapping(n *ast.MappingNode, typ reflect.Type) ast.MapKeyNode {
	switch typ.Kind() {
	case reflect.Map:
		for _, kv := range n.Values {
			if key := firstUnknownKeyIn(kv.Value, typ.Elem()); key != nil {
				return key
			}
		}
	case reflect.Struct:
		fields := map[string]reflect.Type{}
		collectFieldTypes(typ, fields)

		for _, kv := range n.Values {
			if kv.Key.IsMergeKey() {
				continue
			}

			fieldType, ok := fields[keyText(kv.Key)]
			if !ok {
				return kv.Key
			}

			if key := firstUnknownKeyIn(kv.Value, fieldType); key != nil {
				return key
			}
		}
	default:
	}

	return nil
}

// firstUnknownKeyIn is firstUnknownKey for a nested value, skipped when its
// type decodes itself.
func firstUnknownKeyIn(node ast.Node, typ reflect.Type) ast.MapKeyNode {
	if decodesItself(typ) {
		return nil
	}

	return firstUnknownKey(node, typ)
}

// isAfter reports whether token a starts after token b in the source.
func isAfter(a, b *token.Token) bool {
	if a == nil || b == nil || a.Position == nil || b.Position == nil {
		return a != b
	}

	if a.Position.Line != b.Position.Line {
		return a.Position.Line > b.Position.Line
	}

	return a.Position.Column > b.Position.Column
}

// decodesItself reports whether typ, or a pointer to it, has its own
// UnmarshalYAML.
func decodesItself(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	nodeUnmarshaler := reflect.TypeFor[yaml.NodeUnmarshaler]()
	interfaceUnmarshaler := reflect.TypeFor[yaml.InterfaceUnmarshaler]()
	pointer := reflect.PointerTo(typ)

	return pointer.Implements(nodeUnmarshaler) || pointer.Implements(interfaceUnmarshaler)
}

// collectFieldTypes maps the YAML key of every field of struct typ, inline
// fields included, to the field's type, naming fields the way goccy does.
func collectFieldTypes(typ reflect.Type, fields map[string]reflect.Type) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return
	}

	for field := range typ.Fields() {
		tag := field.Tag.Get("yaml")
		if tag == "" {
			tag = field.Tag.Get("json")
		}

		if field.PkgPath != "" && !field.Anonymous || tag == "-" {
			continue
		}

		name, options, _ := strings.Cut(tag, ",")
		if strings.Contains(","+options+",", ",inline,") {
			collectFieldTypes(field.Type, fields)
			continue
		}

		if name == "" {
			name = strings.ToLower(field.Name)
		}
		fields[name] = field.Type
	}
}

// keyText returns a mapping key as the string goccy matches field names with.
func keyText(key ast.MapKeyNode) string {
	if text, ok := ScalarText(key); ok {
		return text
	}

	return key.GetToken().Value
}

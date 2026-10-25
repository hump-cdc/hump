package config

import (
	"strconv"

	"hump/config/decoder"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// sourceScalar is a non-string scalar of the source (a number, a bool, .inf)
// kept as written: decoding it would turn `table: 00000` into 0.
type sourceScalar string

// MarshalYAML writes the scalar back as it was written.
func (s sourceScalar) MarshalYAML() ([]byte, error) {
	return []byte(s), nil
}

// quotedString is a string written as a YAML double-quoted scalar.
type quotedString string

// MarshalYAML writes the string double-quoted, escaping what YAML needs escaped.
func (s quotedString) MarshalYAML() ([]byte, error) {
	return []byte(strconv.Quote(string(s))), nil
}

// sourceValue decodes the first document of raw into the tree expand works on:
// mappings as yaml.MapSlice, sequences as []any, strings as string, null as
// nil and any other scalar as its sourceScalar. Anchors, aliases and merge
// keys are resolved.
func sourceValue(raw string) (any, error) {
	file, err := parser.ParseBytes([]byte(raw), 0)
	if err != nil {
		return nil, err
	}

	if len(file.Docs) == 0 {
		return nil, nil
	}

	return (&sourceDecoder{anchors: map[string]any{}}).value(file.Docs[0].Body), nil
}

// scalarString returns v as a string when it is a string or a sourceScalar,
// and "" otherwise.
func scalarString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case sourceScalar:
		return string(s)
	default:
		return ""
	}
}

// sourceDecoder turns AST nodes into expand's tree, remembering anchors as it
// meets them, which is in document order.
type sourceDecoder struct {
	anchors map[string]any
}

// value returns the tree for node.
func (d *sourceDecoder) value(node ast.Node) any {
	switch n := node.(type) {
	case *ast.AnchorNode:
		v := d.value(n.Value)
		d.anchors[n.Name.GetToken().Value] = v
		return v
	case *ast.AliasNode:
		return d.anchors[n.Value.GetToken().Value]
	case *ast.TagNode:
		if n.Start.Value == "!!str" {
			if text, ok := decoder.ScalarText(n.Value); ok {
				return text
			}
		}
		return d.value(n.Value)
	case *ast.MappingNode:
		return d.mapping(n)
	case *ast.SequenceNode:
		list := make([]any, 0, len(n.Values))
		for _, item := range n.Values {
			list = append(list, d.value(item))
		}
		return list
	case *ast.StringNode:
		return n.Value
	case *ast.LiteralNode:
		return n.Value.Value
	case *ast.NullNode:
		return nil
	case ast.ScalarNode:
		return sourceScalar(n.GetToken().Value)
	default:
		return nil
	}
}

// mapping returns the yaml.MapSlice for n. Keys merged in with `<<` come after
// the mapping's own and never override them; an earlier merge source wins
// over a later one.
func (d *sourceDecoder) mapping(n *ast.MappingNode) yaml.MapSlice {
	out := make(yaml.MapSlice, 0, len(n.Values))
	var merged []yaml.MapSlice

	for _, kv := range n.Values {
		if !kv.Key.IsMergeKey() {
			out = append(out, yaml.MapItem{Key: d.key(kv.Key), Value: d.value(kv.Value)})
			continue
		}

		switch v := d.value(kv.Value).(type) {
		case yaml.MapSlice:
			merged = append(merged, v)
		case []any:
			for _, item := range v {
				if m, ok := item.(yaml.MapSlice); ok {
					merged = append(merged, m)
				}
			}
		default:
		}
	}

	for _, m := range merged {
		out = mergeMapSlice(out, m)
	}

	return out
}

// key returns a mapping key as a string, the way the typed pass matches it.
func (d *sourceDecoder) key(node ast.MapKeyNode) string {
	if text, ok := decoder.ScalarText(node); ok {
		return text
	}

	return node.String()
}

// quoteFragileStrings returns v with every string goccy would not read back as
// itself replaced by its quotedString: goccy writes "\t" bare, which then reads
// back as null.
func quoteFragileStrings(v any) any {
	switch t := v.(type) {
	case yaml.MapSlice:
		out := make(yaml.MapSlice, len(t))
		for i, kv := range t {
			out[i] = yaml.MapItem{Key: kv.Key, Value: quoteFragileStrings(kv.Value)}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = quoteFragileStrings(item)
		}
		return out
	case string:
		if !roundTrips(t) {
			return quotedString(t)
		}
		return t
	default:
		return v
	}
}

// roundTrips reports whether goccy reads s back unchanged from its own output.
func roundTrips(s string) bool {
	out, err := yaml.Marshal(yaml.MapSlice{{Key: "v", Value: s}})
	if err != nil {
		return false
	}

	var back map[string]any
	if err := yaml.Unmarshal(out, &back); err != nil {
		return false
	}

	got, ok := back["v"].(string)

	return ok && got == s
}

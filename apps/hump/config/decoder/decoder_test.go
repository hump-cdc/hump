package decoder

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

func TestDecode(t *testing.T) {
	var out sample
	if err := Decode(parseMapping(t, "name: hump\n"), &out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out.Name != "hump" {
		t.Fatalf("Name = %q, want %q", out.Name, "hump")
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	var out sample
	if err := Decode(parseMapping(t, "name: hump\nextra: 1\n"), &out); err == nil {
		t.Fatalf("expected strict-mode error for unknown field")
	}
}

func TestDecodeRunsValidator(t *testing.T) {
	var out sample
	if err := Decode(parseMapping(t, "other: x\n"), &out); err == nil {
		t.Fatalf("expected validation error for missing required name")
	}
}

func TestKeyValue(t *testing.T) {
	node := parseMapping(t, "a: 1\nb: two\n")

	v := KeyValue(node, "b")
	if v == nil || v.String() != "two" {
		t.Fatalf("KeyValue(b) = %v, want two", v)
	}
	if v := KeyValue(node, "missing"); v != nil {
		t.Fatalf("KeyValue(missing) = %v, want nil", v)
	}
}

func TestNodeLine(t *testing.T) {
	node := parseMapping(t, "a: 1\nb: 2\n")

	tests := []struct {
		name string
		node ast.Node
		want int
	}{
		{name: "nil node", node: nil, want: 0},
		{name: "nil token", node: &ast.NullNode{}, want: 0},
		{name: "nil position", node: &ast.NullNode{Token: &token.Token{}}, want: 0},
		{name: "parsed node", node: KeyValue(node, "b"), want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NodeLine(tt.node); got != tt.want {
				t.Fatalf("NodeLine = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDecodeKeepsScalarTextOfStringFields(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{src: "name: 00000\n", want: "00000"},
		{src: "name: 0x1F\n", want: "0x1F"},
		{src: "name: 1.50\n", want: "1.50"},
		{src: "name: .inf\n", want: ".inf"},
		{src: "name: true\n", want: "true"},
		{src: "name: !!str 007\n", want: "007"},
		{src: "name: \"q\\tx\"\n", want: "q\tx"},
		{src: "name: |-\n  a\n  b\n", want: "a\nb"},
	}

	for _, tt := range tests {
		var out sample
		if err := Decode(parseMapping(t, tt.src), &out); err != nil {
			t.Fatalf("Decode(%q): %v", tt.src, err)
		}
		if out.Name != tt.want {
			t.Errorf("Decode(%q) Name = %q, want %q", tt.src, out.Name, tt.want)
		}
	}
}

func TestDecodeReportsFirstUnknownFieldInDocumentOrder(t *testing.T) {
	node := parseMapping(t, "name: hump\nzz: 1\nnested:\n  name: x\n  yy: 2\naa: 3\nmm: 4\n")

	for range 50 {
		var out nestedSample
		err := Decode(node, &out)
		if err == nil || !strings.Contains(err.Error(), `[2:1] unknown field "zz"`) {
			t.Fatalf("Decode error = %v, want the first unknown field zz", err)
		}
	}
}

func TestUnmarshalReportsFirstUnknownFieldInDocumentOrder(t *testing.T) {
	for range 50 {
		var out nestedSample
		err := Unmarshal([]byte("name: hump\nnested:\n  name: x\n  yy: 2\n  bb: 3\naa: 4\n"), &out)
		if err == nil || !strings.Contains(err.Error(), `[4:3] unknown field "yy"`) {
			t.Fatalf("Unmarshal error = %v, want the first unknown field yy", err)
		}
	}
}

// sample is a minimal decode target with one required field.
type sample struct {
	Name  string `yaml:"name"            validate:"required"`
	Other string `yaml:"other,omitempty"`
}

// nestedSample is a decode target with a nested plain struct.
type nestedSample struct {
	Name   string `yaml:"name"`
	Nested sample `yaml:"nested"`
}

// parseMapping parses src and returns its root mapping node.
func parseMapping(t *testing.T, src string) *ast.MappingNode {
	t.Helper()
	file, err := parser.ParseBytes([]byte(src), 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	node, ok := file.Docs[0].Body.(*ast.MappingNode)
	if !ok {
		t.Fatalf("root is %T, want *ast.MappingNode", file.Docs[0].Body)
	}
	return node
}

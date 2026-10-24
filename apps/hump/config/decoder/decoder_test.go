package decoder

import (
	"reflect"
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

func TestDecodeScalarText(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    string
		wantErr bool
	}{
		{name: "plain scalar", data: "00000", want: "00000"},
		{name: "empty", data: "", want: "keep"},
		{name: "tagged null", data: "!!str null", want: ""},
		{name: "mapping", data: "{a: 1}", want: "keep", wantErr: true},
		{name: "syntax error", data: "a: [", want: "keep", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := "keep"
			err := decodeScalarText(&out, []byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeScalarText(%q) error = %v, wantErr %v", tt.data, err, tt.wantErr)
			}
			if out != tt.want {
				t.Fatalf("decodeScalarText(%q) = %q, want %q", tt.data, out, tt.want)
			}
		})
	}
}

func TestScalarText(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		want   string
		wantOK bool
	}{
		{name: "anchored", src: "v: &a 00000\n", want: "00000", wantOK: true},
		{name: "tagged", src: "v: !!str 7\n", want: "7", wantOK: true},
		{name: "null", src: "v: ~\n", want: "", wantOK: false},
		{name: "mapping", src: "v: {a: 1}\n", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ScalarText(KeyValue(parseMapping(t, tt.src), "v"))
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("ScalarText = %q, %v, want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDecodeReportsFirstUnknownFieldInNestedValues(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "anchored mapping", src: "nested: &n\n  name: x\n  yy: 1\nzz: 2\n", want: `[3:3] unknown field "yy"`},
		{name: "tagged value", src: "anything: !!map\n  a: 1\nzz: 2\nyy: 3\n", want: `[3:1] unknown field "zz"`},
		{name: "sequence item", src: "items:\n  - name: a\n  - name: b\n    yy: 1\nzz: 2\n", want: `[4:5] unknown field "yy"`},
		{name: "map value", src: "byName:\n  a:\n    name: a\n  b:\n    name: b\n    yy: 1\nzz: 2\n", want: `[6:5] unknown field "yy"`},
		{name: "sequence in any", src: "anything: [1, 2]\nzz: 1\nyy: 2\n", want: `[2:1] unknown field "zz"`},
		{name: "json-tagged and untagged fields", src: "jsonName: a\nplain: b\nzz: 1\nyy: 2\n", want: `[3:1] unknown field "zz"`},
		{name: "inline pointer", src: "name: a\nother: b\nzz: 1\nyy: 2\n", want: `[3:1] unknown field "zz"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 20 {
				var out wideSample
				err := Decode(parseMapping(t, tt.src), &out)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("Decode error = %v, want %s", err, tt.want)
				}
			}
		})
	}
}

func TestDecodeSkipsMergeKeyWhenFindingUnknownField(t *testing.T) {
	src := "base: &b\n  name: x\nnested:\n  <<: *b\n  yy: 1\n  zz: 2\n"

	for range 20 {
		var out wideSample
		err := Decode(parseMapping(t, src), &out)
		if err == nil || !strings.Contains(err.Error(), `[5:3] unknown field "yy"`) {
			t.Fatalf("Decode error = %v, want the first unknown field yy", err)
		}
	}
}

func TestIsAfterWithoutPosition(t *testing.T) {
	positioned := &token.Token{Position: &token.Position{Line: 1, Column: 1}}
	tests := []struct {
		name string
		a, b *token.Token
		want bool
	}{
		{name: "both nil", a: nil, b: nil, want: false},
		{name: "one nil", a: positioned, b: nil, want: true},
		{name: "no position", a: &token.Token{}, b: positioned, want: true},
	}

	for _, tt := range tests {
		if got := isAfter(tt.a, tt.b); got != tt.want {
			t.Errorf("isAfter(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCollectFieldTypesOfNonStruct(t *testing.T) {
	fields := map[string]reflect.Type{}
	collectFieldTypes(reflect.TypeFor[*int](), fields)
	if len(fields) != 0 {
		t.Fatalf("collectFieldTypes(*int) = %v, want none", fields)
	}
}

func TestKeyTextOfNonScalarKey(t *testing.T) {
	key := &ast.NullNode{Token: &token.Token{Value: "~"}}
	if got := keyText(key); got != "~" {
		t.Fatalf("keyText(null) = %q, want ~", got)
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

// wideSample is a decode target whose fields cover every shape the
// unknown-field walk descends into.
type wideSample struct {
	*InlineSample `yaml:",inline"`
	Base          sample            `yaml:"base"`
	Nested        sample            `yaml:"nested"`
	Items         []sample          `yaml:"items"`
	ByName        map[string]sample `yaml:"byName"`
	Anything      any               `yaml:"anything"`
	JSONName      string            `json:"jsonName"`
	Plain         string
}

// InlineSample is inlined into wideSample through a pointer; it is exported
// because goccy cannot set an embedded pointer to an unexported type.
type InlineSample struct {
	Name  string `yaml:"name"`
	Other string `yaml:"other"`
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

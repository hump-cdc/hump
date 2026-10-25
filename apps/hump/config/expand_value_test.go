package config

import (
	"reflect"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestSourceValue(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want any
	}{
		{name: "empty document", raw: "", want: nil},
		{name: "no document", raw: "...", want: nil},
		{name: "str tag keeps scalar text", raw: "v: !!str 007\n", want: yaml.MapSlice{{Key: "v", Value: "007"}}},
		{name: "str tag on null", raw: "v: !!str null\n", want: yaml.MapSlice{{Key: "v", Value: nil}}},
		{name: "other tag", raw: "v: !!int 5\n", want: yaml.MapSlice{{Key: "v", Value: sourceScalar("5")}}},
		{name: "null key", raw: "~: 1\n", want: yaml.MapSlice{{Key: "null", Value: sourceScalar("1")}}},
		{name: "merge sequence", raw: "a: &a {x: 1}\nb: &b {x: 2, y: 3}\nc:\n  <<: [*a, *b, 4]\n",
			want: yaml.MapSlice{
				{Key: "a", Value: yaml.MapSlice{{Key: "x", Value: sourceScalar("1")}}},
				{Key: "b", Value: yaml.MapSlice{{Key: "x", Value: sourceScalar("2")}, {Key: "y", Value: sourceScalar("3")}}},
				{Key: "c", Value: yaml.MapSlice{{Key: "x", Value: sourceScalar("1")}, {Key: "y", Value: sourceScalar("3")}}},
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sourceValue(tt.raw)
			if err != nil {
				t.Fatalf("sourceValue(%q): %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("sourceValue(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSourceValue_SyntaxError(t *testing.T) {
	if _, err := sourceValue("a: ["); err == nil {
		t.Fatal("sourceValue: expected a syntax error")
	}
}

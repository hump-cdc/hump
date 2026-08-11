package config

import (
	pkgconfig "hump/pkg/config"

	"github.com/goccy/go-yaml/ast"
)

var Validate = pkgconfig.Validate

// decode decodes node into out with strict mode and the same validator wired
// into the outer decoder, so validation failures surface as token-anchored
// yaml.Error values.
func decode(node ast.Node, out any) error {
	return pkgconfig.Decode(node, out)
}

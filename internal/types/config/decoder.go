package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

var Validate = validator.New(validator.WithRequiredStructEnabled())

// decode decodes node into out with strict mode and the same validator wired
// into the outer decoder, so validation failures surface as token-anchored
// yaml.Error values.
func decode(node ast.Node, out any) error {
	return yaml.NodeToValue(node, out, yaml.Strict(), yaml.Validator(Validate))
}

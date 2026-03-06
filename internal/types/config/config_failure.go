package config

import (
	"fmt"

	"github.com/goccy/go-yaml/ast"
)

type FailureMode string

const (
	FailureModeCrash FailureMode = "crash"
	FailureModeSkip  FailureMode = "skip"
)

type Failure struct {
	Mode FailureMode `yaml:"mode,omitempty" validate:"omitempty,oneof=crash skip"`
}

func (f *Failure) UnmarshalYAML(node ast.Node) error {
	if scalar, ok := node.(ast.ScalarNode); ok {
		value, ok := scalar.GetValue().(string)
		if !ok {
			return fmt.Errorf("line %d: failure: must be a string or mapping", nodeLine(node))
		}
		f.Mode = FailureMode(value)
		if err := Validate.Struct(f); err != nil {
			return err
		}
	} else {
		type failure Failure
		if err := decode(node, (*failure)(f)); err != nil {
			return err
		}
	}

	f.setDefaults()

	return nil
}

func (f *Failure) setDefaults() {
	if f.Mode == "" {
		f.Mode = FailureModeCrash
	}
}

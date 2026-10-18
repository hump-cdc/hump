package resilience

import (
	"fmt"

	"hump/config/decoder"
	"hump/pkg/resilience"

	"github.com/goccy/go-yaml/ast"
)

// Failure is the exhausted-retries policy of a sink. Every delivery is
// attached — the LSN advances only once the sink completes — so the configured
// mode always takes effect: the event a [resilience.FailureModeCrash] sink
// stops the process on is still recoverable on restart.
type Failure struct {
	Mode resilience.FailureMode `yaml:"mode,omitempty" validate:"omitempty,oneof=crash skip"`
}

// UnmarshalYAML accepts either a bare mode string or a mapping, then fills in
// the default mode.
func (f *Failure) UnmarshalYAML(node ast.Node) error {
	if scalar, ok := node.(ast.ScalarNode); ok {
		value, ok := scalar.GetValue().(string)
		if !ok {
			return fmt.Errorf("line %d: failure: must be a string or mapping", decoder.NodeLine(node))
		}
		f.Mode = resilience.FailureMode(value)
		if err := decoder.Validate.Struct(f); err != nil {
			return err
		}
	} else {
		type failure Failure
		if err := decoder.Decode(node, (*failure)(f)); err != nil {
			return err
		}
	}

	f.SetDefaults()

	return nil
}

// SetDefaults sets the mode to crash when none is configured.
func (f *Failure) SetDefaults() {
	if f.Mode == "" {
		f.Mode = resilience.FailureModeCrash
	}
}

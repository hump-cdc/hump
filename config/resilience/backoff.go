// Package resilience holds the per-sink delivery policies of the configuration:
// what to do on exhausted retries, how to retry, and how to back off.
package resilience

import (
	"time"

	"hump/config/decoder"
	"hump/pkg/resilience"

	"github.com/goccy/go-yaml/ast"
)

// Backoff is the wire format of the inter-attempt delay policy. It mirrors
// [resilience.Backoff] field for field — same names, types and order — so
// [Backoff.Policy] converts to the runtime policy without copying members one
// by one. Keep the two declarations in step.
type Backoff struct {
	Type       resilience.BackoffType `yaml:"type"                  validate:"omitempty,oneof=linear exponential"`
	Initial    time.Duration          `yaml:"initial,omitempty"     validate:"omitempty,gt=0"`
	Max        time.Duration          `yaml:"max,omitempty"         validate:"omitempty,gt=0"`
	Increment  time.Duration          `yaml:"increment,omitempty"   validate:"omitempty,gt=0"`            // linear only
	Multiplier float64                `yaml:"multiplier,omitempty"  validate:"omitempty,gt=0"`            // exponential only
	Jitter     resilience.JitterMode  `yaml:"jitter,omitempty"      validate:"omitempty,oneof=full none"` // exponential only
}

// Policy returns the runtime backoff policy this configuration describes.
func (b Backoff) Policy() resilience.Backoff {
	return resilience.Backoff(b)
}

// UnmarshalYAML decodes a backoff policy strictly and fills in its defaults.
func (b *Backoff) UnmarshalYAML(node ast.Node) error {
	type backoff Backoff
	if err := decoder.Decode(node, (*backoff)(b)); err != nil {
		return err
	}

	b.setDefaults()

	return nil
}

func (b *Backoff) setDefaults() {
	if b.Type == "" {
		b.Type = resilience.BackoffTypeExponential
	}

	if b.Initial <= 0 {
		b.Initial = 1 * time.Second
	}

	if b.Max <= 0 {
		b.Max = 30 * time.Second
	}

	if b.Multiplier <= 0 {
		b.Multiplier = 2
	}

	if b.Jitter == "" {
		b.Jitter = resilience.JitterModeFull
	}
}

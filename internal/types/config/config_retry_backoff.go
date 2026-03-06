package config

import (
	"time"

	"github.com/goccy/go-yaml/ast"
)

type BackoffType string

const (
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeExponential BackoffType = "exponential"
)

type JitterMode string

const (
	JitterModeFull JitterMode = "full"
	JitterModeNone JitterMode = "none"
)

type Backoff struct {
	Type       BackoffType   `yaml:"type"                  validate:"omitempty,oneof=linear exponential"`
	Initial    time.Duration `yaml:"initial,omitempty"     validate:"omitempty,gt=0"`
	Max        time.Duration `yaml:"max,omitempty"         validate:"omitempty,gt=0"`
	Increment  time.Duration `yaml:"increment,omitempty"   validate:"omitempty,gt=0"`            // linear only
	Multiplier float64       `yaml:"multiplier,omitempty"  validate:"omitempty,gt=0"`            // exponential only
	Jitter     JitterMode    `yaml:"jitter,omitempty"      validate:"omitempty,oneof=full none"` // exponential only
}

func (b *Backoff) UnmarshalYAML(node ast.Node) error {
	type backoff Backoff
	if err := decode(node, (*backoff)(b)); err != nil {
		return err
	}

	b.setDefaults()

	return nil
}

func (b *Backoff) setDefaults() {
	if b.Type == "" {
		b.Type = BackoffTypeExponential
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
		b.Jitter = JitterModeFull
	}
}

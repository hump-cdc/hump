package config

import (
	"time"

	"github.com/goccy/go-yaml/ast"
)

type Retry struct {
	MaxAttempts    int           `yaml:"maxAttempts,omitempty"    validate:"omitempty,gt=0"`
	AttemptTimeout time.Duration `yaml:"attemptTimeout,omitempty" validate:"omitempty,gt=0"`
	Backoff        Backoff       `yaml:"backoff,omitempty"`
}

func (r *Retry) UnmarshalYAML(node ast.Node) error {
	type retry Retry
	if err := decode(node, (*retry)(r)); err != nil {
		return err
	}

	r.setDefaults()

	return nil
}

func (r *Retry) setDefaults() {
	if r.MaxAttempts <= 0 {
		r.MaxAttempts = 5
	}

	if r.AttemptTimeout <= 0 {
		r.AttemptTimeout = 10 * time.Second
	}

	r.Backoff.setDefaults()
}

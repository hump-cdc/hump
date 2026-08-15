package resilience

import (
	"time"

	"hump/config/decoder"

	"github.com/goccy/go-yaml/ast"
)

type Retry struct {
	MaxAttempts    int           `yaml:"maxAttempts,omitempty"    validate:"omitempty,gt=0"`
	AttemptTimeout time.Duration `yaml:"attemptTimeout,omitempty" validate:"omitempty,gt=0"`
	Backoff        Backoff       `yaml:"backoff,omitempty"`
}

func (r *Retry) UnmarshalYAML(node ast.Node) error {
	type retryPolicy Retry
	if err := decoder.Decode(node, (*retryPolicy)(r)); err != nil {
		return err
	}

	r.SetDefaults()

	return nil
}

func (r *Retry) SetDefaults() {
	if r.MaxAttempts <= 0 {
		r.MaxAttempts = 5
	}

	if r.AttemptTimeout <= 0 {
		r.AttemptTimeout = 10 * time.Second
	}

	r.Backoff.setDefaults()
}

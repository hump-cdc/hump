package resilience

import (
	"time"

	"hump/config/decoder"

	"github.com/goccy/go-yaml/ast"
)

// Retry is the retry policy of a sink: how many attempts, how long each may
// take, and the delay between them.
type Retry struct {
	MaxAttempts    int           `yaml:"maxAttempts,omitempty"    validate:"omitempty,gt=0"`
	AttemptTimeout time.Duration `yaml:"attemptTimeout,omitempty" validate:"omitempty,gt=0"`
	Backoff        Backoff       `yaml:"backoff,omitempty"`
}

// UnmarshalYAML decodes a retry policy strictly and fills in its defaults.
func (r *Retry) UnmarshalYAML(node ast.Node) error {
	type retryPolicy Retry
	if err := decoder.Decode(node, (*retryPolicy)(r)); err != nil {
		return err
	}

	r.SetDefaults()

	return nil
}

// SetDefaults fills in 5 attempts, a 10s attempt timeout and the backoff
// defaults wherever they are unset.
func (r *Retry) SetDefaults() {
	if r.MaxAttempts <= 0 {
		r.MaxAttempts = 5
	}

	if r.AttemptTimeout <= 0 {
		r.AttemptTimeout = 10 * time.Second
	}

	r.Backoff.setDefaults()
}

package health

import (
	"context"
	"errors"
	"sync/atomic"

	hg "github.com/hellofresh/health-go/v5"
)

// healthCheckState is the atomically swappable snapshot held by a
// healthCheck: value reports whether the component is currently
// operational, and err carries the last observed failure (may be nil).
type healthCheckState struct {
	value bool
	err   error
}

// healthCheck is a named probe whose state is updated from the observed
// component and read by the health-go check function on every HTTP request.
// State transitions are lock-free via atomic.Pointer.
type healthCheck struct {
	name  string
	state atomic.Pointer[healthCheckState]
}

// check returns a health-go option that registers this probe under its
// name. When skipOnErr is true, the aggregate handler treats a failure as
// non-fatal (used for the liveness probe to ignore transient WAL outages).
func (c *healthCheck) check(skipOnErr bool) hg.Option {
	return hg.WithChecks(hg.Config{
		Name:      c.name,
		SkipOnErr: skipOnErr,
		Check: func(ctx context.Context) error {
			state := c.state.Load()
			if !state.value {
				if state.err != nil {
					return state.err
				} else {
					return errors.New("not operational")
				}
			}

			return nil
		},
	})
}

// Operational reports whether the probe's most recent update marked the
// component as operational. A probe that has never been updated reports false.
func (c *healthCheck) Operational() bool {
	state := c.state.Load()
	return state != nil && state.value
}

// Update atomically replaces the probe state. value reports whether the
// observed component is operational, and err is the last observed error
// (may be nil). When value is false and err is nil, the probe reports a
// generic "not operational" failure.
func (c *healthCheck) Update(value bool, err error) {
	c.state.Store(&healthCheckState{
		value: value,
		err:   err,
	})
}

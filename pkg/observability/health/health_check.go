package health

import (
	"context"
	"sync/atomic"

	hg "github.com/hellofresh/health-go/v5"
)

// state is one snapshot of a check, swapped atomically.
type state struct {
	operational bool
	err         error
}

// Check is one health flag a component sets as it gains or loses its
// connection.
type Check struct {
	name  string
	state atomic.Pointer[state]
}

// newCheck creates a check named name that starts unhealthy.
func newCheck(name string) *Check {
	hc := &Check{
		name: name,
	}
	hc.state.Store(&state{operational: false})

	return hc
}

// configure registers the check with health-go.
func (hc *Check) configure() hg.Option {
	return hg.WithChecks(hg.Config{
		Name: hc.name,
		Check: func(_ context.Context) error {
			state := hc.state.Load()
			if !state.operational {
				if state.err != nil {
					return state.err
				}

				return ErrNotOperational
			}

			return nil
		},
	})
}

// MarkHealthy reports the component as operational.
func (hc *Check) MarkHealthy() {
	hc.state.Store(&state{
		operational: true,
		err:         nil,
	})
}

// MarkUnhealthy reports the component as down, with err as the cause.
func (hc *Check) MarkUnhealthy(err error) {
	hc.state.Store(&state{
		operational: false,
		err:         err,
	})
}

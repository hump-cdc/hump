package health

import (
	"context"
	"sync/atomic"

	hg "github.com/hellofresh/health-go/v5"
)

type state struct {
	operational bool
	err         error
}

type HealthCheck struct {
	name  string
	state atomic.Pointer[state]
}

func newHealthCheck(name string) *HealthCheck {
	hc := &HealthCheck{
		name: name,
	}
	hc.state.Store(&state{operational: false})

	return hc
}

func (hc *HealthCheck) configure() hg.Option {
	return hg.WithChecks(hg.Config{
		Name: hc.name,
		Check: func(ctx context.Context) error {
			state := hc.state.Load()
			if !state.operational {
				if state.err != nil {
					return state.err
				} else {
					return ErrNotOperational
				}
			}

			return nil
		},
	})
}

func (hc *HealthCheck) MarkHealthy() {
	hc.state.Store(&state{
		operational: true,
		err:         nil,
	})
}

func (hc *HealthCheck) MarkUnhealthy(err error) {
	hc.state.Store(&state{
		operational: false,
		err:         err,
	})
}

// Package stopsnapshot stops the snapshot run in progress, or clears a
// finished one back to idle.
package stopsnapshot

import (
	"context"
	"errors"

	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2"
)

// Controller serves POST /api/snapshot/stop.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller stopping snapshots through useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Handler returns the huma handler stopping the run in progress, or clearing
// a finished one.
func (c *Controller) Handler() func(ctx context.Context, _ *struct{}) (*struct{}, error) {
	return func(_ context.Context, _ *struct{}) (*struct{}, error) {
		err := c.useCase.Stop()
		switch {
		case errors.Is(err, snapshot.ErrNotRunning):
			return nil, huma.Error409Conflict(err.Error())
		case errors.Is(err, ErrUnavailable):
			return nil, huma.Error503ServiceUnavailable(err.Error())
		case err != nil:
			return nil, err
		}

		return nil, nil
	}
}

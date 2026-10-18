package stopsnapshot

import (
	"errors"

	"hump/internal/port/snapshot"
)

// ErrUnavailable is returned when the server runs no snapshot runner.
var ErrUnavailable = errors.New("snapshot is unavailable")

// UseCase stops or clears snapshot runs.
type UseCase struct {
	runner snapshot.Runner // runner is nil when only the API is described, never served.
}

// NewUseCase returns a use case stopping snapshot runs on runner.
func NewUseCase(runner snapshot.Runner) *UseCase {
	return &UseCase{
		runner: runner,
	}
}

// Stop ends the snapshot in progress or clears a finished one, or returns
// snapshot.ErrNotRunning when the runner is idle.
func (uc *UseCase) Stop() error {
	if uc.runner == nil {
		return ErrUnavailable
	}

	return uc.runner.Stop()
}

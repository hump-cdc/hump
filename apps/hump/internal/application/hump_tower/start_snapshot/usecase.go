package startsnapshot

import (
	"errors"

	"hump/internal/port/snapshot"
)

// ErrUnavailable is returned when the server runs no snapshot runner.
var ErrUnavailable = errors.New("snapshot is unavailable")

// UseCase starts snapshot runs.
type UseCase struct {
	runner snapshot.Runner // runner is nil when only the API is described, never served.
}

// NewUseCase returns a use case starting snapshot runs on runner.
func NewUseCase(runner snapshot.Runner) *UseCase {
	return &UseCase{
		runner: runner,
	}
}

// Start begins a snapshot of tables, or returns snapshot.ErrAlreadyRunning.
func (uc *UseCase) Start(tables []snapshot.Request) error {
	if uc.runner == nil {
		return ErrUnavailable
	}

	return uc.runner.Start(tables)
}

package streamsnapshot

import (
	"hump/internal/port/snapshot"
)

type UseCase struct {
	runner snapshot.Runner // runner is nil when only the API is described, never served.
}

// NewUseCase returns a use case watching the snapshot runs of runner.
func NewUseCase(runner snapshot.Runner) *UseCase {
	return &UseCase{
		runner: runner,
	}
}

// Subscribe attaches one client: it returns the current state, the channel
// later states arrive on and the func that detaches the client. Without a
// runner the state is idle and the channel nil, which blocks forever.
func (uc *UseCase) Subscribe() (snapshot.State, <-chan snapshot.State, func()) {
	if uc.runner == nil {
		return snapshot.State{Status: snapshot.StatusIdle}, nil, func() {}
	}

	id, state, updates := uc.runner.Subscribe()

	return state, updates, func() { uc.runner.Unsubscribe(id) }
}

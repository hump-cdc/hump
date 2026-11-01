package startsnapshot

import (
	"errors"

	"hump/config"
	"hump/internal/port/snapshot"
)

// ErrUnavailable is returned when the server runs no snapshot runner.
var ErrUnavailable = errors.New("snapshot is unavailable")

// ErrTableNotConfigured reports a target that cannot resolve to a configured pair.
var ErrTableNotConfigured = errors.New("snapshot table is not configured or is ambiguous")

// UseCase starts snapshot runs.
type UseCase struct {
	cfg    config.Configuration
	runner snapshot.Runner // runner is nil when only the API is described, never served.
}

// NewUseCase returns a use case starting configured snapshot targets on runner.
func NewUseCase(cfg config.Configuration, runner snapshot.Runner) *UseCase {
	return &UseCase{
		cfg:    cfg,
		runner: runner,
	}
}

// Start resolves the entire batch before starting a snapshot without mutating tables.
func (uc *UseCase) Start(tables []snapshot.Request) error {
	if uc.runner == nil {
		return ErrUnavailable
	}

	resolved := make([]snapshot.Request, 0, len(tables))
	for _, request := range tables {
		table, ok := uc.cfg.ResolveSnapshotTable(request.Table)
		if !ok {
			return ErrTableNotConfigured
		}
		resolved = append(resolved, snapshot.Request{Table: table, Where: request.Where})
	}

	return uc.runner.Start(resolved)
}

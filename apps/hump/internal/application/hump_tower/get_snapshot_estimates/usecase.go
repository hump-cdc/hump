package getsnapshotestimates

import (
	"context"
	"errors"
	"sync"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

// defaultMaxConcurrentEstimates caps the tables counted at once; each holds a
// database session of its own while it runs.
const defaultMaxConcurrentEstimates = 6

// ErrTableNotConfigured reports a target that cannot resolve to a configured pair.
var ErrTableNotConfigured = errors.New("snapshot table is not configured or is ambiguous")

// Request is one table to count and the filter its rows are counted with.
type Request struct {
	Table domain.TableIdentifier
	Where string
}

// Estimate is one row count of one table, or the error that prevented it.
type Estimate struct {
	Table domain.TableIdentifier
	Count uint64
	Rough bool
	Error error
}

// UseCase counts the rows a snapshot of the requested tables would read.
type UseCase struct {
	cfg                    config.Configuration
	db                     database.Database // db is nil when only the API is described, never served.
	maxConcurrentEstimates int
	slots                  chan struct{}
}

// Option configures a UseCase.
type Option func(*UseCase)

// WithMaxConcurrentEstimates sets how many tables are counted at once. A
// non-positive n leaves the default.
func WithMaxConcurrentEstimates(n int) Option {
	return func(uc *UseCase) {
		if n > 0 {
			uc.maxConcurrentEstimates = n
		}
	}
}

// NewUseCase returns a use case counting configured snapshot targets in db.
func NewUseCase(cfg config.Configuration, db database.Database, opts ...Option) *UseCase {
	uc := &UseCase{
		cfg:                    cfg,
		db:                     db,
		maxConcurrentEstimates: defaultMaxConcurrentEstimates,
	}

	for _, opt := range opts {
		opt(uc)
	}
	uc.slots = make(chan struct{}, uc.maxConcurrentEstimates)

	return uc
}

// Estimate counts every requested table in a goroutine of its own, at most
// uc.maxConcurrentEstimates across all calls, and streams their estimates until
// all counts close. A cancelled ctx stops delivery and queued counts.
func (uc *UseCase) Estimate(ctx context.Context, requests []Request) <-chan Estimate {
	estimates := make(chan Estimate)

	if uc.db == nil {
		close(estimates)
		return estimates
	}

	var wg sync.WaitGroup
	for _, request := range requests {
		wg.Go(func() {
			select {
			case uc.slots <- struct{}{}:
				defer func() { <-uc.slots }()
			case <-ctx.Done():
				return
			}

			// select picks at random when a slot frees up after ctx is done.
			if ctx.Err() != nil {
				return
			}

			uc.estimateTable(ctx, request, estimates)
		})
	}

	go func() {
		wg.Wait()
		close(estimates)
	}()

	return estimates
}

// estimateTable sends the row counts of one table to estimates, or the error
// that prevented them, and waits for GetRowCount to finish even after cancellation.
func (uc *UseCase) estimateTable(ctx context.Context, request Request, estimates chan<- Estimate) {
	emit := func(estimate Estimate) bool {
		select {
		case estimates <- estimate:
			return true
		case <-ctx.Done():
			return false
		}
	}

	resolved, ok := uc.cfg.ResolveSnapshotTable(request.Table)
	if !ok {
		emit(Estimate{Table: request.Table, Error: ErrTableNotConfigured})
		return
	}

	counts, err := uc.db.GetRowCount(ctx, resolved, request.Where)
	if err != nil {
		emit(Estimate{Table: request.Table, Error: err})
		return
	}

	for count := range counts {
		if !emit(Estimate{Table: request.Table, Count: count.Num, Rough: count.Rough, Error: count.Error}) {
			// Keep the slot until the adapter has closed its database session.
			for range counts {
			}
			return
		}
	}
}

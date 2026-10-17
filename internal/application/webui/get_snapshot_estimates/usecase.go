package getsnapshotestimates

import (
	"context"
	"sync"

	"hump/internal/domain"
	"hump/internal/port/database"
)

// defaultMaxConcurrentEstimates caps the tables counted at once; each holds a
// database session of its own while it runs.
const defaultMaxConcurrentEstimates = 6

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

type UseCase struct {
	db                     database.Database // db is nil when only the API is described, never served.
	maxConcurrentEstimates int
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

// NewUseCase returns a use case counting rows of db.
func NewUseCase(db database.Database, opts ...Option) *UseCase {
	uc := &UseCase{
		db:                     db,
		maxConcurrentEstimates: defaultMaxConcurrentEstimates,
	}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}

// Estimate counts every requested table in a goroutine of its own, at most
// uc.maxConcurrentEstimates at a time, and streams their estimates on a channel
// that closes once all of them are done. A cancelled ctx stops them early.
func (uc *UseCase) Estimate(ctx context.Context, requests []Request) <-chan Estimate {
	estimates := make(chan Estimate)

	if uc.db == nil {
		close(estimates)
		return estimates
	}

	slots := make(chan struct{}, uc.maxConcurrentEstimates)

	var wg sync.WaitGroup
	for _, request := range requests {
		wg.Go(func() {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
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
// that prevented them, until GetRowCount is done or ctx is cancelled.
func (uc *UseCase) estimateTable(ctx context.Context, request Request, estimates chan<- Estimate) {
	emit := func(estimate Estimate) bool {
		select {
		case estimates <- estimate:
			return true
		case <-ctx.Done():
			return false
		}
	}

	counts, err := uc.db.GetRowCount(ctx, request.Table, request.Where)
	if err != nil {
		emit(Estimate{Table: request.Table, Error: err})
		return
	}

	for count := range counts {
		if !emit(Estimate{Table: request.Table, Count: count.Num, Rough: count.Rough, Error: count.Error}) {
			return
		}
	}
}

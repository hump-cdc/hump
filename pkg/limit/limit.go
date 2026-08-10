package limit

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"

	"hump/internal/telemetry/metrics"
)

// Limit bounds concurrency per key with a lazily created semaphore per key.
type Limit struct {
	mu       sync.Mutex
	limiters map[string]*semaphore.Weighted
	metrics  *metrics.Metrics
}

func New(metrics *metrics.Metrics) *Limit {
	return &Limit{
		limiters: make(map[string]*semaphore.Weighted),
		metrics:  metrics,
	}
}

// Acquire blocks until a slot is available for the given key, or ctx is done.
// The semaphore for the key is created lazily on first use with the provided
// concurrency value; subsequent calls reuse the existing semaphore regardless
// of the concurrency argument.
func (l *Limit) Acquire(ctx context.Context, key string, concurrency int64) error {
	sem := l.getSemaphore(key, concurrency)

	l.metrics.LimiterPendingInc(key)
	if err := sem.Acquire(ctx, 1); err != nil {
		l.metrics.LimiterPendingDec(key) // never acquired; caller won't Release
		return err
	}
	return nil
}

// Release frees one slot previously acquired for the given key. Safe to call
// only when a matching Acquire succeeded.
func (l *Limit) Release(key string) {
	l.mu.Lock()
	sem := l.limiters[key]
	l.mu.Unlock()

	if sem != nil {
		sem.Release(1)
		l.metrics.LimiterPendingDec(key)
	}
}

func (l *Limit) getSemaphore(key string, concurrency int64) *semaphore.Weighted {
	l.mu.Lock()
	defer l.mu.Unlock()

	if sem, ok := l.limiters[key]; ok {
		return sem
	}

	if concurrency < 1 {
		concurrency = 1
	}

	sem := semaphore.NewWeighted(concurrency)
	l.limiters[key] = sem

	return sem
}

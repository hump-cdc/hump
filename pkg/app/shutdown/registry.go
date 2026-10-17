package shutdown

import (
	"context"
	"errors"
	"hump/pkg/observability/log"
	"slices"
	"sync"
	"time"
)

var logger = log.Scope("shutdown")

// Holds the ordered set of shutdown handlers and ensures shutdown
// runs at most once.
type registry struct {
	mu       sync.Mutex
	once     sync.Once
	handlers []shutdownFunc
}

// Pairs a handler with its stage and a human-readable name used in logs.
type shutdownFunc struct {
	stage Stage
	name  string
	fn    func(context.Context) error
}

// Appends a named handler for stage to the registry. Safe for concurrent use.
func (s *registry) add(stage Stage, name string, fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = append(s.handlers, shutdownFunc{
		stage: stage,
		name:  name,
		fn:    fn,
	})
}

// Runs all registered handlers stage by stage and, within a stage, in reverse
// registration order, each bounded by timeout.
func (s *registry) shutdown(timeout time.Duration) error {
	logger.Info("Shutting down...")

	var result error
	s.once.Do(func() {
		s.mu.Lock()
		handlers := s.handlers
		s.handlers = nil
		s.mu.Unlock()

		if len(handlers) == 0 {
			return
		}

		var fnErrors []error

		for stage := range stageCount {
			for _, h := range slices.Backward(handlers) {
				if h.stage != stage {
					continue
				}

				if err := run(h, timeout); err != nil {
					fnErrors = append(fnErrors, err)
				}
			}
		}

		result = errors.Join(fnErrors...)
	})

	return result
}

// run calls h bounded by timeout and returns its error, or ErrShutdownTimeout
// when it does not finish in time.
func run(h shutdownFunc, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := make(chan error, 1)
	start := time.Now()
	go func() {
		logger.Debug("Stopping down service...", "name", h.name)
		result <- h.fn(ctx)
	}()

	select {
	case err := <-result:
		if err != nil {
			logger.Error("Stop service error", "name", h.name, "duration", time.Since(start), "error", err)
		} else {
			logger.Info("Service stopped", "name", h.name, "duration", time.Since(start))
		}

		return err
	case <-ctx.Done():
		logger.Warn("Timeout to stop service", "name", h.name, "duration", time.Since(start))

		return ErrShutdownTimeout
	}
}

package shutdown

import (
	"context"
	"errors"
	"hump/pkg/telemetry/log"
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

// Pairs a handler with a human-readable name used in logs.
type shutdownFunc struct {
	name string
	fn   func(context.Context) error
}

// Appends a named handler to the registry. Safe for concurrent use.
func (s *registry) add(name string, fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = append(s.handlers, shutdownFunc{
		name: name,
		fn:   fn,
	})
}

// Runs all registered handlers in reverse registration order, each
// bounded by timeout.
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

		for _, h := range slices.Backward(handlers) {
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
					fnErrors = append(fnErrors, err)
					logger.Error("Stop service error", "name", h.name, "duration", time.Since(start))
				} else {
					logger.Info("Service stopped", "name", h.name, "duration", time.Since(start))
				}
			case <-ctx.Done():
				fnErrors = append(fnErrors, ErrShutdownTimeout)
				logger.Warn("Timeout to stop service", "name", h.name, "duration", time.Since(start))
			}
		}

		result = errors.Join(fnErrors...)
	})

	return result
}

package retry

import (
	"errors"
	"slices"
	"sync"
)

// Errors is a goroutine-safe accumulator of errors. The zero value is ready
// to use.
type Errors struct {
	mu     sync.Mutex
	errors []error
}

// Append adds err to the accumulator. It is safe to call concurrently.
func (e *Errors) Append(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = append(e.errors, err)
}

// Snapshot returns a copy of the currently accumulated errors. The returned
// slice is independent of further mutations to e.
func (e *Errors) Snapshot() []error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return slices.Clone(e.errors)
}

// Error returns the accumulated errors joined into a single error, or nil
// when none have been appended.
func (e *Errors) Error() error {
	errs := e.Snapshot()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

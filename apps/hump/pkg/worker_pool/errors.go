package workerpool

import (
	"errors"
	"fmt"
)

var (
	// ErrAlreadyInFlight is returned by BeginBatch while a batch is open.
	ErrAlreadyInFlight = errors.New("worker_pool: batch is already started")
	// ErrNotInFlight is returned by Submit and FlushBatch without an open batch.
	ErrNotInFlight = errors.New("worker_pool: batch is not started yet")
	// ErrWorkerPoolClosed is returned once the pool has shut down.
	ErrWorkerPoolClosed = errors.New("worker_pool: worker pool is closed")
	// ErrEmptyTask is returned by Submit for a nil context or function.
	ErrEmptyTask = errors.New("worker_pool: task context or fn is nil")
	// ErrTaskDropped is returned by Submit when BackpressureDrop discards a task.
	ErrTaskDropped = errors.New("worker_pool: task dropped due to backpressure")
	// ErrWorkerFailed is returned by FlushBatch when any task of the batch failed.
	ErrWorkerFailed = errors.New("worker_pool: some of the tasks in the batch failed")
)

// WorkerPanicError is a task panic recovered by a worker, with its stack.
type WorkerPanicError struct {
	Recovered any
	Stack     []byte
}

// Error describes the recovered panic value.
func (e *WorkerPanicError) Error() string {
	return fmt.Sprintf("worker_pool job panic: %v", e.Recovered)
}

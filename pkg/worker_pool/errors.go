package workerpool

import (
	"errors"
	"fmt"
)

var (
	ErrAlreadyInFlight  = errors.New("worker_pool: batch is already started")
	ErrNotInFlight      = errors.New("worker_pool: batch is not started yet")
	ErrWorkerPoolClosed = errors.New("worker_pool: worker pool is closed")
	ErrEmptyTask        = errors.New("worker_pool: task context or fn is nil")
	ErrTaskDropped      = errors.New("worker_pool: task dropped due to backpressure")
	ErrWorkerFailed     = errors.New("worker_pool: some of the tasks in the batch failed")
)

type WorkerPanicError struct {
	Recovered any
	Stack     []byte
}

func (e *WorkerPanicError) Error() string {
	return fmt.Sprintf("worker_pool job panic: %v", e.Recovered)
}

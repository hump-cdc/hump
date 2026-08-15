package workerpool

import (
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// WorkerPool runs tasks on a reusable pool of goroutines and keeps them aligned to
// batches: a batch of tasks is scheduled, then awaited, before the next one starts.
//
// T is the payload type every task of the pool receives: Submit takes the payload by
// value and the worker hands it to the task function, so per-task data does not have
// to be captured in a closure.
//
// Expected usage is New, then BeginBatch -> Submit... -> FlushBatch (the hot path) from a
// single goroutine, since the batch API is not thread-safe; Shutdown is called separately.
type WorkerPool[T any] struct {
	mu sync.Mutex

	inflight atomic.Bool
	closed   bool

	workerWg sync.WaitGroup
	taskWg   sync.WaitGroup

	tasks              chan task[T]
	droppedTasks       atomic.Uint64
	backpressurePolicy BackpressurePolicy

	errors        chan error
	hasError      atomic.Bool
	droppedErrors atomic.Uint64
}

// New starts a worker pool with the given options and returns it ready to accept batches.
// The payload type has to be given explicitly, as in New[Payload](WithWorkers(4)).
// The spawned goroutines live until Shutdown is called.
func New[T any](opts ...Option) *WorkerPool[T] {
	config := newConfig(opts...)
	wp := &WorkerPool[T]{
		tasks:              make(chan task[T], config.queueSize),
		errors:             make(chan error, config.errorBuf),
		backpressurePolicy: config.backpressurePolicy,
	}

	wp.workerWg.Add(config.workers)
	for range config.workers {
		go wp.worker()
	}

	return wp
}

// BeginBatch opens a new batch and must always be paired with FlushBatch.
// Returns ErrAlreadyInFlight if a batch is open already.
func (wp *WorkerPool[T]) BeginBatch() error {
	if !wp.inflight.CompareAndSwap(false, true) {
		return ErrAlreadyInFlight
	}

	wp.mu.Lock()

	if wp.closed {
		wp.inflight.Store(false)
		wp.mu.Unlock()
		return ErrWorkerPoolClosed
	}

	wp.hasError.Store(false)

	return nil
}

// Submit queues fn for execution inside the current batch without waiting for it, and
// hands payload to fn when it runs.
// Depending on the backpressure policy a full queue either blocks or returns ErrTaskDropped.
func (wp *WorkerPool[T]) Submit(ctx context.Context, fn func(ctx context.Context, payload T) error, payload T) error {
	if ctx == nil || fn == nil {
		return ErrEmptyTask
	}

	if !wp.inflight.Load() {
		return ErrNotInFlight
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t := task[T]{ctx: ctx, fn: fn, payload: payload}

	wp.taskWg.Add(1)
	if wp.backpressurePolicy == BackpressureBlock {
		// Block. The queue usually has room, so try a plain non-blocking send first:
		// it avoids the runtime select machinery on the common path.
		select {
		case wp.tasks <- t:
			return nil
		default:
		}

		// The queue is full. A context that can never be cancelled needs no select either.
		done := ctx.Done()
		if done == nil {
			wp.tasks <- t
			return nil
		}

		select {
		case wp.tasks <- t:
			return nil
		case <-done:
			wp.taskWg.Done()
			return ctx.Err()
		}
	} else {
		// Drop
		select {
		case wp.tasks <- t:
			return nil
		default:
			wp.taskWg.Done()
			wp.droppedTasks.Add(1)
			return ErrTaskDropped
		}
	}
}

// FlushBatch waits for all tasks of the current batch to finish and closes the batch.
// It returns ErrWorkerFailed if any task of the batch failed, or nil if all of them succeeded.
// The individual errors are available on the Errors channel.
func (wp *WorkerPool[T]) FlushBatch() error {
	if !wp.inflight.CompareAndSwap(true, false) {
		return ErrNotInFlight
	}

	wp.taskWg.Wait()

	var err error
	if wp.hasError.Load() {
		err = ErrWorkerFailed
	}

	wp.mu.Unlock()

	return err
}

// Shutdown gracefully stops the pool: it waits for the current batch to be flushed, stops
// the workers and closes the Errors channel.
// It is meant to be called from a separate goroutine and returns ErrWorkerPoolClosed if repeated.
func (wp *WorkerPool[T]) Shutdown() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		return ErrWorkerPoolClosed
	}
	wp.closed = true
	close(wp.tasks)

	wp.workerWg.Wait()

	close(wp.errors)

	return nil
}

// DroppedTasks returns the total number of tasks rejected because the queue was full
// under the BackpressureDrop policy.
func (wp *WorkerPool[T]) DroppedTasks() uint64 {
	return wp.droppedTasks.Load()
}

// DroppedErrors returns the total number of errors discarded because the Errors channel
// was full at the moment of reporting.
func (wp *WorkerPool[T]) DroppedErrors() uint64 {
	return wp.droppedErrors.Load()
}

// Errors returns the channel with errors reported by tasks. Reporting is non-blocking, so
// an error that cannot be delivered is dropped and DroppedErrors is incremented instead.
func (wp *WorkerPool[T]) Errors() <-chan error {
	return wp.errors
}

func (wp *WorkerPool[T]) worker() {
	defer wp.workerWg.Done()

	for task := range wp.tasks {
		wp.runTask(&task)
	}
}

func (wp *WorkerPool[T]) runTask(t *task[T]) {
	defer func() {
		if r := recover(); r != nil {
			wp.reportError(&WorkerPanicError{Recovered: r, Stack: debug.Stack()})
		}

		wp.taskWg.Done()
	}()

	if err := t.ctx.Err(); err != nil {
		wp.reportError(err)
		return
	}

	if err := t.fn(t.ctx, t.payload); err != nil {
		wp.reportError(err)
	}
}

func (wp *WorkerPool[T]) reportError(err error) {
	if err == nil {
		return
	}

	wp.hasError.Store(true)

	select {
	case wp.errors <- err:
	default:
		wp.droppedErrors.Add(1)
	}
}

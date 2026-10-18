# pkg/worker_pool — how this package hangs together

A fixed set of goroutines that runs work in **batches**: open a batch, submit tasks, wait for all of
them. Its one user is `process_operation.UseCase`, where a batch is "one row, delivered to every
sink of its table" — the row is not acknowledged until the batch is flushed.

The directory is `worker_pool`, the package is `workerpool`.

## Layout

| File | Role |
|---|---|
| `worker_pool.go` | `WorkerPool[T]`: `New`, the batch API, `Shutdown`, the workers |
| `config.go` | the options, their defaults and their validation |
| `backpressure.go` | `BackpressurePolicy` |
| `errors.go` | the sentinel errors and `WorkerPanicError` |
| `task.go` | the queued unit: context, function, payload |

## The batch API belongs to one goroutine

`BeginBatch → Submit… → FlushBatch`, in that order, from a single goroutine. It is not thread-safe
and is not meant to become so:

- A second `BeginBatch` while a batch is open does **not** wait; it returns `ErrAlreadyInFlight`.
- `Submit` checks only that *a* batch is open, so a concurrent caller's tasks would land in someone
  else's batch and be awaited by their `FlushBatch`.

Callers that can be reached from more than one goroutine serialise themselves.
`adapter/snapshot` holds `processMu` around every `process` call for exactly this reason, and WAL
and snapshot never overlap because a snapshot pauses WAL delivery.

`T` is the payload every task receives. `Submit` takes it by value alongside the function, so the
hot path (`process_operation`) passes `binding.fn`, a function built once per binding and
stored in a struct field, plus a `*sink.Context`, instead of allocating a closure per task.

## `mu` is held across the whole batch

`BeginBatch` locks `wp.mu` and returns with it **held**; `FlushBatch` unlocks it. That is the
mechanism behind shutdown, not an oversight:

- `Shutdown` takes the same mutex, so it parks until the open batch is flushed and can never close
  the task channel under a `Submit`. Tasks submitted while it is already waiting still run
  (`TestWorkerPool_shutdown_runsSubmitsArrivingWhileItWaits`).
- A `BeginBatch` without its `FlushBatch` therefore blocks `Shutdown` forever — and with it the
  `StageWorkers` hook, until the registry's timeout gives up on it.
- `Shutdown` must come from a goroutine other than the one driving batches.

After `Shutdown`, `BeginBatch` returns `ErrWorkerPoolClosed`, and `Submit`/`FlushBatch` return
`ErrNotInFlight`. `process_operation` returns `BeginBatch`'s error wrapped as `begin batch: …` and
submits nothing. A second `Shutdown` returns `ErrWorkerPoolClosed`.

## Errors come back two ways

`FlushBatch` answers one question — did anything in this batch fail — with `ErrWorkerFailed` or nil.
The flag behind it is reset by the next `BeginBatch`.

The errors themselves go to `Errors()`, a buffered channel written **without blocking**: when it is
full the error is counted in `DroppedErrors` and lost, while the batch still reports the failure.
Somebody has to drain that channel for the errors to be seen at all; `process_operation` logs them
from a goroutine that ends when `Shutdown` closes the channel.

What counts as a task error:

- the function returning one;
- a **panic**, recovered into `*WorkerPanicError` with the stack — the worker survives;
- the task's context already being done when a worker picks it up. The function is not called, the
  context's error goes to `Errors()`, and `FlushBatch` returns `ErrWorkerFailed`.

## Backpressure

`BackpressureBlock`, the default, makes `Submit` wait for queue room, giving up only if the task's
context ends. `BackpressureDrop` returns `ErrTaskDropped` and counts it in `DroppedTasks`. The
pipeline uses `Block`: dropping there would lose a row.

`Submit` under `Block` tries a plain non-blocking send before falling into a `select`, and skips the
`select` entirely for a context that can never be cancelled. Both are measured fast paths; keep them
when touching that function.

## Configuration panics

`New` validates its options and **panics** on a bad one — it does not return an error:

- workers must be positive;
- the queue must be at least as large as the worker count;
- the error buffer must be non-negative;
- the backpressure policy must be `BackpressureBlock` or `BackpressureDrop`.

The defaults are 10 workers and a queue of 10, so `WithWorkers(n)` with `n > 10` panics unless
`WithQueueSize` is raised with it. `process_operation` sets both to the largest number of sinks any
table has.

## Tests

`background` runs a call on a goroutine and returns its result channel; `awaitErr` reads it with a
deadline. `gate` is a task that reports it started and then blocks until released. `drainErrors`
collects `Errors()` in the background and returns a snapshot function plus a wait.
`sinkTask`/`failingSinkTask` simulate a sink write of a given duration.

`TestWorkerPool_shutdown_concurrentWithBatchLoop` fires `Shutdown` at fifty different offsets into a
batch loop and requires that no task is lost and nothing hangs; it is the test to run under `-race`
after changing the locking.

`worker_pool_bench_test.go` benchmarks the pool, and `worker_pool_baseline_bench_test.go` runs the
same workload on a bare `WaitGroup`, an `errgroup` and a semaphore for comparison.

Helpers live below the last test function — see the `go` skill. The exceptions are in
`worker_pool_bench_test.go`: the `sinkCounts` and `taskDurations` vars and the `benchPayload` type,
declared above `BenchmarkWorkerPool`.

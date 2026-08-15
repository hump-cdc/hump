package workerpool

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_beginBatch(t *testing.T) {
	wp := New[struct{}]()
	defer wp.Shutdown()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}

	if err := wp.BeginBatch(); !errors.Is(err, ErrAlreadyInFlight) {
		t.Errorf("got %v, want ErrAlreadyInFlight", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	// Batches are reusable: the pool takes a new one once the previous is flushed.
	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Errorf("second FlushBatch: got %v, want nil", err)
	}
}

func TestWorkerPool_beginBatch_afterShutdown(t *testing.T) {
	wp := New[struct{}]()

	if err := wp.Shutdown(); err != nil {
		t.Fatalf("Shutdown: got %v, want nil", err)
	}

	for attempt := range 2 {
		if err := wp.BeginBatch(); !errors.Is(err, ErrWorkerPoolClosed) {
			t.Errorf("attempt %d: got %v, want ErrWorkerPoolClosed", attempt, err)
		}
	}
}

func TestWorkerPool_flushBatch_notInFlight(t *testing.T) {
	wp := New[struct{}]()
	defer wp.Shutdown()

	if err := wp.FlushBatch(); !errors.Is(err, ErrNotInFlight) {
		t.Errorf("without a batch: got %v, want ErrNotInFlight", err)
	}

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); !errors.Is(err, ErrNotInFlight) {
		t.Errorf("after a flush: got %v, want ErrNotInFlight", err)
	}
}

func TestWorkerPool_submit_invalid(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Millisecond))
	t.Cleanup(cancelExpired)

	tests := map[string]struct {
		ctx  context.Context
		fn   func(ctx context.Context, payload struct{}) error
		want error
	}{
		"nil context":          {ctx: nil, fn: sinkTask(0), want: ErrEmptyTask},
		"nil task":             {ctx: context.Background(), fn: nil, want: ErrEmptyTask},
		"nil context and task": {ctx: nil, fn: nil, want: ErrEmptyTask},
		"cancelled context":    {ctx: cancelled, fn: sinkTask(0), want: context.Canceled},
		"expired deadline":     {ctx: expired, fn: sinkTask(0), want: context.DeadlineExceeded},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			wp := New[struct{}]()
			defer wp.Shutdown()

			if err := wp.BeginBatch(); err != nil {
				t.Fatalf("BeginBatch: got %v, want nil", err)
			}

			if err := wp.Submit(test.ctx, test.fn, struct{}{}); !errors.Is(err, test.want) {
				t.Errorf("got %v, want %v", err, test.want)
			}

			wp.FlushBatch()
		})
	}
}

func TestWorkerPool_submit_notInFlight(t *testing.T) {
	wp := New[struct{}]()
	defer wp.Shutdown()

	ctx := context.Background()

	if err := wp.Submit(ctx, sinkTask(0), struct{}{}); !errors.Is(err, ErrNotInFlight) {
		t.Errorf("before BeginBatch: got %v, want ErrNotInFlight", err)
	}

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}

	if err := wp.Submit(ctx, sinkTask(0), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	if err := wp.Submit(ctx, sinkTask(0), struct{}{}); !errors.Is(err, ErrNotInFlight) {
		t.Errorf("after FlushBatch: got %v, want ErrNotInFlight", err)
	}
}

func TestWorkerPool_flushBatch_waitsForTasks(t *testing.T) {
	// taskPayload is what the pool carries for a task: the number of the task it
	// was submitted with.
	type taskPayload struct {
		n int
	}

	tests := map[string]struct {
		workers   int
		queueSize int
		tasks     int
	}{
		"single worker":            {workers: 1, queueSize: 1, tasks: 50},
		"default sized pool":       {workers: 10, queueSize: 10, tasks: 100},
		"queue smaller than batch": {workers: 4, queueSize: 4, tasks: 1000},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			wp := New[taskPayload](WithWorkers(test.workers), WithQueueSize(test.queueSize))
			defer wp.Shutdown()

			var done atomic.Int32

			// The payload carries the number of the task submitted with it, so
			// seen[n] tells whether that payload reached its task, and only it.
			seen := make([]atomic.Int32, test.tasks)
			ctx := context.Background()

			if err := wp.BeginBatch(); err != nil {
				t.Fatalf("BeginBatch: got %v, want nil", err)
			}
			for n := range test.tasks {
				if err := wp.Submit(ctx, func(_ context.Context, payload taskPayload) error {
					done.Add(1)
					seen[payload.n].Add(1)
					return nil
				}, taskPayload{n: n}); err != nil {
					t.Fatalf("Submit: got %v, want nil", err)
				}
			}

			if err := wp.FlushBatch(); err != nil {
				t.Fatalf("FlushBatch: got %v, want nil", err)
			}

			if got, want := done.Load(), int32(test.tasks); got != want {
				t.Errorf("got %d tasks done, want %d", got, want)
			}

			for n := range seen {
				if got := seen[n].Load(); got != 1 {
					t.Errorf("payload %d: got it %d times, want exactly once", n, got)
				}
			}
		})
	}
}

func TestWorkerPool_flushBatch_returnsErrWorkerFailed(t *testing.T) {
	wp := New[struct{}]()
	defer wp.Shutdown()

	ctx := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	for range 4 {
		if err := wp.Submit(ctx, sinkTask(0), struct{}{}); err != nil {
			t.Fatalf("Submit: got %v, want nil", err)
		}
	}
	if err := wp.Submit(ctx, failingSinkTask(0), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); !errors.Is(err, ErrWorkerFailed) {
		t.Errorf("got %v, want ErrWorkerFailed", err)
	}

	select {
	case err := <-wp.Errors():
		if !errors.Is(err, errTask) {
			t.Errorf("Errors(): got %v, want errTask", err)
		}
	default:
		t.Error("got no error on Errors(), want errTask")
	}
}

func TestWorkerPool_flushBatch_clearsError(t *testing.T) {
	wp := New[struct{}]()
	defer wp.Shutdown()

	ctx := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(ctx, failingSinkTask(0), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); !errors.Is(err, ErrWorkerFailed) {
		t.Fatalf("first batch: got %v, want ErrWorkerFailed", err)
	}

	// The error belongs to the batch that produced it, not to the pool.
	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(ctx, sinkTask(0), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Errorf("second batch: got %v, want nil", err)
	}
}

func TestWorkerPool_submit_contextCancelledInQueue(t *testing.T) {
	wp := New[struct{}](WithWorkers(1), WithQueueSize(1), WithErrorBufferSize(1))
	defer wp.Shutdown()

	started, release := make(chan struct{}), make(chan struct{})
	bg := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(bg, gate(started, release), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}
	<-started

	ctx, cancel := context.WithCancel(bg)

	var ran atomic.Bool
	if err := wp.Submit(ctx, func(context.Context, struct{}) error {
		ran.Store(true)
		return nil
	}, struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	cancel()
	close(release)

	if err := wp.FlushBatch(); !errors.Is(err, ErrWorkerFailed) {
		t.Errorf("FlushBatch: got %v, want ErrWorkerFailed", err)
	}

	if ran.Load() {
		t.Error("got the queued task executed, want it skipped after its context was cancelled")
	}

	select {
	case err := <-wp.Errors():
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Errors(): got %v, want context.Canceled", err)
		}
	default:
		t.Error("got no error on Errors(), want context.Canceled")
	}
}

func TestWorkerPool_errors_deliversTaskErrors(t *testing.T) {
	const tasks = 5

	wp := New[struct{}](WithErrorBufferSize(tasks))
	getCapturedErrors, waitForErrorsDrain := drainErrors(wp)

	ctx := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	for range tasks {
		if err := wp.Submit(ctx, failingSinkTask(0), struct{}{}); err != nil {
			t.Fatalf("Submit: got %v, want nil", err)
		}
	}

	if err := wp.FlushBatch(); !errors.Is(err, ErrWorkerFailed) {
		t.Fatalf("FlushBatch: got %v, want ErrWorkerFailed", err)
	}

	if err := wp.Shutdown(); err != nil {
		t.Fatalf("Shutdown: got %v, want nil", err)
	}

	waitForErrorsDrain()
	seen := getCapturedErrors()
	if got, want := len(seen), tasks; got != want {
		t.Fatalf("got %d errors, want %d", got, want)
	}

	for i, err := range seen {
		if !errors.Is(err, errTask) {
			t.Errorf("error %d: got %v, want errTask", i, err)
		}
	}

	if got := wp.DroppedErrors(); got != 0 {
		t.Errorf("got %d dropped errors, want 0", got)
	}
}

func TestWorkerPool_droppedErrors(t *testing.T) {
	const tasks = 5

	wp := New[struct{}](WithErrorBufferSize(0))
	defer wp.Shutdown()

	ctx := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	for range tasks {
		if err := wp.Submit(ctx, failingSinkTask(0), struct{}{}); err != nil {
			t.Fatalf("Submit: got %v, want nil", err)
		}
	}

	if err := wp.FlushBatch(); !errors.Is(err, ErrWorkerFailed) {
		t.Errorf("FlushBatch: got %v, want ErrWorkerFailed", err)
	}

	if got, want := wp.DroppedErrors(), uint64(tasks); got != want {
		t.Errorf("got %d dropped errors, want %d", got, want)
	}
}

func TestWorkerPool_errors_closedOnShutdown(t *testing.T) {
	wp := New[struct{}]()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(context.Background(), sinkTask(0), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	if err := wp.Shutdown(); err != nil {
		t.Fatalf("Shutdown: got %v, want nil", err)
	}

	select {
	case err, ok := <-wp.Errors():
		if ok {
			t.Errorf("got error %v, want a closed channel", err)
		}
	case <-time.After(time.Second):
		t.Error("got a blocked Errors() channel, want it closed by Shutdown")
	}
}

func TestWorkerPool_panicRecovered(t *testing.T) {
	wp := New[struct{}](WithErrorBufferSize(1))
	defer wp.Shutdown()

	ctx := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(ctx, func(context.Context, struct{}) error { panic("boom") }, struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); !errors.Is(err, ErrWorkerFailed) {
		t.Fatalf("FlushBatch: got %v, want ErrWorkerFailed", err)
	}

	var panicErr *WorkerPanicError
	select {
	case err := <-wp.Errors():
		if !errors.As(err, &panicErr) {
			t.Fatalf("Errors(): got %T (%v), want *WorkerPanicError", err, err)
		}
	default:
		t.Fatal("got no error on Errors(), want *WorkerPanicError")
	}

	if got, want := panicErr.Recovered, "boom"; got != want {
		t.Errorf("got recovered %v, want %q", got, want)
	}

	if len(panicErr.Stack) == 0 {
		t.Error("got an empty stack, want the stack of the panicking task")
	}

	if got, want := panicErr.Error(), "worker_pool job panic: boom"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The worker that recovered the panic must stay in the pool.
	var ran atomic.Bool

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(ctx, func(context.Context, struct{}) error {
		ran.Store(true)
		return nil
	}, struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("second batch: got %v, want nil", err)
	}

	if !ran.Load() {
		t.Error("got a wedged pool after a panic, want it still running tasks")
	}
}

func TestWorkerPool_submit_dropPolicy(t *testing.T) {
	wp := New[struct{}](WithWorkers(1), WithQueueSize(1), WithBackpressurePolicy(BackpressureDrop))
	defer wp.Shutdown()

	started, release := make(chan struct{}), make(chan struct{})
	ctx := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(ctx, gate(started, release), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}
	<-started

	if err := wp.Submit(ctx, sinkTask(0), struct{}{}); err != nil { // fills the only queue slot
		t.Fatalf("Submit: got %v, want nil", err)
	}

	var ran atomic.Bool
	dropped := wp.Submit(ctx, func(context.Context, struct{}) error {
		ran.Store(true)
		return nil
	}, struct{}{})

	if !errors.Is(dropped, ErrTaskDropped) {
		t.Fatalf("got %v, want ErrTaskDropped", dropped)
	}

	close(release)

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	if got, want := wp.DroppedTasks(), uint64(1); got != want {
		t.Errorf("got %d dropped tasks, want %d", got, want)
	}

	if ran.Load() {
		t.Error("got the dropped task executed, want it never queued")
	}
}

func TestWorkerPool_submit_blockPolicy_blocks(t *testing.T) {
	cancellable := t.Context()

	tests := map[string]struct {
		ctx context.Context
	}{
		"context that cannot be cancelled": {ctx: context.Background()},
		"cancellable context":              {ctx: cancellable},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			wp := New[struct{}](WithWorkers(1), WithQueueSize(1))
			defer wp.Shutdown()

			started, release := make(chan struct{}), make(chan struct{})
			ctx := test.ctx

			var done atomic.Int64
			count := func(context.Context, struct{}) error {
				done.Add(1)
				return nil
			}

			if err := wp.BeginBatch(); err != nil {
				t.Fatalf("BeginBatch: got %v, want nil", err)
			}
			if err := wp.Submit(ctx, gate(started, release), struct{}{}); err != nil {
				t.Fatalf("Submit: got %v, want nil", err)
			}
			<-started

			if err := wp.Submit(ctx, count, struct{}{}); err != nil { // fills the only queue slot
				t.Fatalf("Submit: got %v, want nil", err)
			}

			blocked := background(func() error {
				return wp.Submit(ctx, count, struct{}{})
			})

			select {
			case err := <-blocked:
				t.Fatalf("Submit on a full queue returned %v before it was unblocked", err)
			case <-time.After(50 * time.Millisecond):
			}

			close(release)

			if err := awaitErr(t, blocked, 50*time.Millisecond, "Submit on a full queue"); err != nil {
				t.Fatalf("got %v, want nil", err)
			}

			if err := wp.FlushBatch(); err != nil {
				t.Fatalf("FlushBatch: got %v, want nil", err)
			}

			if got, want := done.Load(), int64(2); got != want {
				t.Errorf("got %d tasks done, want %d", got, want)
			}

			if got := wp.DroppedTasks(); got != 0 {
				t.Errorf("got %d dropped tasks, want 0", got)
			}
		})
	}
}

func TestWorkerPool_submit_blockPolicy_cancelWhileBlocked(t *testing.T) {
	wp := New[struct{}](WithWorkers(1), WithQueueSize(1))
	defer wp.Shutdown()

	started, release := make(chan struct{}), make(chan struct{})
	bg := context.Background()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}
	if err := wp.Submit(bg, gate(started, release), struct{}{}); err != nil {
		t.Fatalf("Submit: got %v, want nil", err)
	}
	<-started

	if err := wp.Submit(bg, sinkTask(0), struct{}{}); err != nil { // fills the only queue slot
		t.Fatalf("Submit: got %v, want nil", err)
	}

	ctx, cancel := context.WithCancel(bg)

	blocked := background(func() error { return wp.Submit(ctx, sinkTask(0), struct{}{}) })
	select {
	case err := <-blocked:
		t.Fatalf("Submit on a full queue returned %v before it was unblocked", err)
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	if err := awaitErr(t, blocked, time.Second, "cancelled Submit"); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}

	close(release)

	if err := wp.FlushBatch(); err != nil {
		t.Errorf("FlushBatch: got %v, want nil", err)
	}
}

func TestWorkerPool_shutdown_idempotent(t *testing.T) {
	wp := New[struct{}]()

	if err := wp.Shutdown(); err != nil {
		t.Fatalf("first Shutdown: got %v, want nil", err)
	}

	if err := wp.Shutdown(); !errors.Is(err, ErrWorkerPoolClosed) {
		t.Errorf("second Shutdown: got %v, want ErrWorkerPoolClosed", err)
	}
}

func TestWorkerPool_shutdown_blockedByOpenBatch(t *testing.T) {
	wp := New[struct{}]()

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}

	shutdown := background(wp.Shutdown)
	select {
	case err := <-shutdown:
		t.Fatalf("Shutdown with an open batch returned %v before it was unblocked", err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	if err := awaitErr(t, shutdown, time.Second, "Shutdown"); err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

func TestWorkerPool_shutdown_runsSubmitsArrivingWhileItWaits(t *testing.T) {
	const tasks = 200

	wp := New[struct{}](WithWorkers(4), WithQueueSize(4))

	if err := wp.BeginBatch(); err != nil {
		t.Fatalf("BeginBatch: got %v, want nil", err)
	}

	// Shutdown parks until the open batch is flushed.
	shutdown := background(wp.Shutdown)
	select {
	case err := <-shutdown:
		t.Fatalf("Shutdown with an open batch returned %v before it was unblocked", err)
	case <-time.After(50 * time.Millisecond):
	}

	// Tasks submitted while Shutdown is already waiting still have to run.
	var done atomic.Int32
	ctx := context.Background()

	for range tasks {
		if err := wp.Submit(ctx, func(context.Context, struct{}) error {
			done.Add(1)
			return nil
		}, struct{}{}); err != nil {
			t.Fatalf("Submit while Shutdown waits: got %v, want nil", err)
		}
	}

	if err := wp.FlushBatch(); err != nil {
		t.Fatalf("FlushBatch: got %v, want nil", err)
	}

	if got := done.Load(); got != tasks {
		t.Errorf("got %d tasks done, want %d", got, tasks)
	}

	if err := awaitErr(t, shutdown, time.Second, "Shutdown"); err != nil {
		t.Errorf("Shutdown: got %v, want nil", err)
	}

	if err := wp.BeginBatch(); !errors.Is(err, ErrWorkerPoolClosed) {
		t.Errorf("BeginBatch after Shutdown: got %v, want ErrWorkerPoolClosed", err)
	}
}

func TestWorkerPool_shutdown_concurrentWithBatchLoop(t *testing.T) {
	// The hot path is a loop of BeginBatch -> Submit... -> FlushBatch on one
	// goroutine, while Shutdown fires from another one at an arbitrary moment.
	// Whatever the interleaving, no submitted task may be lost and no call may hang.
	const (
		rounds        = 50
		tasksPerBatch = 20
	)

	for round := range rounds {
		wp := New[struct{}](WithWorkers(4), WithQueueSize(4))
		_, waitErrors := drainErrors(wp)

		// Vary the moment Shutdown lands relative to the batch loop.
		delay := time.Duration(round) * 20 * time.Microsecond
		shutdown := background(func() error {
			time.Sleep(delay)
			return wp.Shutdown()
		})

		var done, submitted atomic.Int32
		ctx := context.Background()
		task := func(context.Context, struct{}) error {
			done.Add(1)
			return nil
		}

		deadline := time.Now().Add(5 * time.Second)
		for {
			if time.Now().After(deadline) {
				t.Fatalf("round %d: batch loop made no progress within 5s", round)
			}

			err := wp.BeginBatch()
			if errors.Is(err, ErrWorkerPoolClosed) {
				break
			}
			if err != nil {
				t.Fatalf("round %d: BeginBatch: got %v, want nil", round, err)
			}

			for range tasksPerBatch {
				if err := wp.Submit(ctx, task, struct{}{}); err != nil {
					t.Fatalf("round %d: Submit: got %v, want nil", round, err)
				}
				submitted.Add(1)
			}

			if err := wp.FlushBatch(); err != nil {
				t.Fatalf("round %d: FlushBatch: got %v, want nil", round, err)
			}

			// FlushBatch has returned, so every task of this batch is accounted for.
			if got, want := done.Load(), submitted.Load(); got != want {
				t.Fatalf("round %d: got %d tasks done, want %d", round, got, want)
			}
		}

		if err := awaitErr(t, shutdown, 5*time.Second, "Shutdown"); err != nil {
			t.Fatalf("round %d: Shutdown: got %v, want nil", round, err)
		}

		if got, want := done.Load(), submitted.Load(); got != want {
			t.Errorf("round %d: got %d tasks done, want %d", round, got, want)
		}

		waitErrors()
	}
}

var errTask = errors.New("task failed")

func sinkTask(d time.Duration) func(context.Context, struct{}) error {
	if d == 0 {
		return func(context.Context, struct{}) error {
			return nil
		}
	}

	return func(context.Context, struct{}) error {
		time.Sleep(d)
		return nil
	}
}

func failingSinkTask(d time.Duration) func(context.Context, struct{}) error {
	if d == 0 {
		return func(context.Context, struct{}) error {
			return errTask
		}
	}

	return func(context.Context, struct{}) error {
		time.Sleep(d)
		return errTask
	}
}

func background(fn func() error) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- fn() }()
	return ch
}

func gate(started chan<- struct{}, release <-chan struct{}) func(context.Context, struct{}) error {
	return func(context.Context, struct{}) error {
		close(started)
		<-release
		return nil
	}
}

func awaitErr(t *testing.T, ch <-chan error, d time.Duration, what string) error {
	t.Helper()

	select {
	case err := <-ch:
		return err
	case <-time.After(d):
		t.Fatalf("%s hung: no return within %s", what, d)
		return nil
	}
}

func drainErrors[T any](wp *WorkerPool[T]) (snapshot func() []error, wait func()) {
	var mu sync.Mutex
	var seen []error

	done := make(chan struct{})
	go func() {
		defer close(done)
		for err := range wp.Errors() {
			mu.Lock()
			seen = append(seen, err)
			mu.Unlock()
		}
	}()

	snapshot = func() []error {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}

	return snapshot, func() { <-done }
}

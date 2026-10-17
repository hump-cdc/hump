package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitSignalsDrainBeforeJoiningWorkers(t *testing.T) {
	r, cancel := newTestRunner(t)

	drained := make(chan struct{})
	r.drain = func() { close(drained) }

	r.Run(func(context.Context) error {
		<-drained
		return nil
	})

	cancel()

	if err := r.Wait(); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
}

func TestWaitJoinsWorkersBeforeShutdownHooks(t *testing.T) {
	r, cancel := newTestRunner(t)

	var finished atomic.Bool
	r.Run(func(ctx context.Context) error {
		<-ctx.Done()
		time.Sleep(20 * time.Millisecond)
		finished.Store(true)

		return ctx.Err()
	})

	finishedAtShutdown := false
	r.shutdown = func(time.Duration) error {
		finishedAtShutdown = finished.Load()
		return nil
	}

	cancel()

	if err := r.Wait(); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}

	if !finishedAtShutdown {
		t.Error("shutdown hooks ran before the worker finished")
	}
}

func TestWaitJoinsServersAfterShutdownHooks(t *testing.T) {
	r, cancel := newTestRunner(t)

	stopped := make(chan struct{})
	var returned atomic.Bool
	r.Serve(func(context.Context) error {
		<-stopped
		time.Sleep(20 * time.Millisecond)
		returned.Store(true)

		return nil
	})

	r.shutdown = func(time.Duration) error {
		close(stopped)
		return nil
	}

	cancel()

	if err := r.Wait(); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}

	if !returned.Load() {
		t.Error("Wait() returned before the server did")
	}
}

func TestWaitReturnsWorkerErrorAfterSignal(t *testing.T) {
	r, cancel := newTestRunner(t)

	wantErr := errors.New("sink unavailable")
	r.Run(func(ctx context.Context) error {
		<-ctx.Done()
		return wantErr
	})

	cancel()

	if err := r.Wait(); !errors.Is(err, wantErr) {
		t.Fatalf("Wait() error = %v, want %v", err, wantErr)
	}
}

func TestWaitReturnsWorkerErrorThatStoppedRunner(t *testing.T) {
	r, _ := newTestRunner(t)

	wantErr := errors.New("stream failed")
	r.Run(func(context.Context) error {
		return wantErr
	})

	if err := r.Wait(); !errors.Is(err, wantErr) {
		t.Fatalf("Wait() error = %v, want %v", err, wantErr)
	}
}

func TestWaitIgnoresRequestedShutdown(t *testing.T) {
	r, _ := newTestRunner(t)

	r.Run(func(context.Context) error {
		return ErrShutdown
	})

	if err := r.Wait(); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
}

func TestWaitReportsDrainTimeout(t *testing.T) {
	r, cancel := newTestRunner(t)
	r.drainTimeout = 10 * time.Millisecond

	block := make(chan struct{})
	t.Cleanup(func() { close(block) })

	r.Run(func(context.Context) error {
		<-block
		return nil
	})

	shutdownCalled := false
	r.shutdown = func(time.Duration) error {
		shutdownCalled = true
		return nil
	}

	cancel()

	if err := r.Wait(); !errors.Is(err, ErrDrainTimeout) {
		t.Fatalf("Wait() error = %v, want ErrDrainTimeout", err)
	}

	if !shutdownCalled {
		t.Error("shutdown hooks did not run after the drain timed out")
	}
}

func TestWaitGivesUpOnStuckServers(t *testing.T) {
	r, cancel := newTestRunner(t)
	r.hookTimeout = 10 * time.Millisecond

	block := make(chan struct{})
	t.Cleanup(func() { close(block) })

	r.Serve(func(context.Context) error {
		<-block
		return nil
	})

	cancel()

	if err := r.Wait(); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
}

// newTestRunner returns a runner detached from the global shutdown registry,
// with short timeouts, and the function that cancels its parent context.
func newTestRunner(t *testing.T) (*Runner, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	r, release := New(ctx)
	t.Cleanup(release)

	r.drainTimeout = time.Second
	r.hookTimeout = time.Second
	r.drain = func() {}
	r.shutdown = func(time.Duration) error { return nil }

	return r, cancel
}

package app

import (
	"context"
	"errors"
	"hump/pkg/app/shutdown"
	"hump/pkg/telemetry/log"
	"sync"
	"syscall"
	"time"
)

var logger = log.Scope("app")

const (
	// drainTimeout caps how long Wait lets Run functions finish in-flight work.
	drainTimeout = 30 * time.Second
	// hookTimeout caps each shutdown hook.
	hookTimeout = 5 * time.Second
)

type runner struct {
	context context.Context
	cancel  context.CancelCauseFunc
	release context.CancelFunc

	// drainTimeout and hookTimeout are the package consts outside tests, which
	// shorten them.
	drainTimeout time.Duration
	hookTimeout  time.Duration
	// drain and shutdown are the shutdown package functions outside tests,
	// which swap them to keep off the global registry.
	drain    func()
	shutdown func(timeout time.Duration) error

	// workers tracks Run functions, servers tracks Serve functions.
	workers sync.WaitGroup
	servers sync.WaitGroup

	mu   sync.Mutex
	errs []error
}

// New creates a runner whose context is cancelled on SIGINT/SIGTERM or when
// any function it runs returns an error. The returned CancelFunc releases the
// OS signal hook and should be deferred by the caller.
func New(ctx context.Context) (*runner, context.CancelFunc) {
	ctx, release := signals(ctx, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancelCause(ctx)

	return &runner{
		release:      release,
		context:      ctx,
		cancel:       cancel,
		drainTimeout: drainTimeout,
		hookTimeout:  hookTimeout,
		drain:        shutdown.Drain,
		shutdown:     shutdown.Shutdown,
	}, release
}

// Run launches each worker function in its own goroutine. Wait lets workers
// finish before any shutdown hook runs, so their context being cancelled means
// "take no new work", not "abort".
func (r *runner) Run(fns ...func(context.Context) error) {
	r.spawn(&r.workers, fns)
}

// Serve launches each server function in its own goroutine. Servers are
// stopped by their StageServers hooks and joined after them.
func (r *runner) Serve(fns ...func(context.Context) error) {
	r.spawn(&r.servers, fns)
}

// Wait blocks until the runner's context is done (OS signal or error), then
// drains workers, runs shutdown hooks, joins servers and returns every error
// the functions reported, apart from shutdown causes.
func (r *runner) Wait() error {
	<-r.context.Done()

	// Restore default signal handling, so a second signal kills the process.
	r.release()

	// Let workers finish in-flight work without starting anything new.
	r.drain()
	if !waitTimeout(&r.workers, r.drainTimeout) {
		logger.Warn("Timeout to drain in-flight work", "timeout", r.drainTimeout)
		r.record(ErrDrainTimeout)
	}

	// Stop worker pools, then sinks, then servers.
	_ = r.shutdown(r.hookTimeout)

	// Servers return once their hooks closed them; a server that never got to
	// register a hook must not hang the exit.
	if !waitTimeout(&r.servers, r.hookTimeout) {
		logger.Warn("Timeout to join servers", "timeout", r.hookTimeout)
	}

	return r.result()
}

// spawn runs each of fns in a goroutine tracked by wg. A returned error is
// recorded and cancels the runner's context with it as the cause.
func (r *runner) spawn(wg *sync.WaitGroup, fns []func(context.Context) error) {
	for _, fn := range fns {
		wg.Go(func() {
			if err := fn(r.context); err != nil {
				r.record(err)
				r.cancel(err)
			}
		})
	}
}

// record keeps err for Wait to return.
func (r *runner) record(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.errs = append(r.errs, err)
}

// result joins the recorded errors, dropping the ones that only restate a
// requested shutdown.
func (r *runner) result() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	errs := make([]error, 0, len(r.errs))
	for _, err := range r.errs {
		if errors.Is(err, context.Canceled) || errors.Is(err, ErrShutdown) {
			continue
		}

		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// waitTimeout waits for wg and reports whether it finished within timeout.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

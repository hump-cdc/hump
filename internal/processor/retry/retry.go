// Package retry runs sink handlers with configurable retry, backoff, timeout,
// and async concurrency control, recording per-sink metrics.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"hump/internal/log"
	asynclimiter "hump/internal/processor/async_limiter"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/config"
	"hump/internal/types/errs"
)

// retry executes sink handlers with retry, backoff, and concurrency control,
// recording metrics and accumulating fatal errors for crash-on-failure sinks.
type retry struct {
	wg           *sync.WaitGroup
	asyncLimiter *asynclimiter.AsyncLimiter
	logger       *log.Log
	metrics      *metrics.Metrics
	errors       *Errors
}

func New(wg *sync.WaitGroup, traceparent string, asyncLimiter *asynclimiter.AsyncLimiter, metrics *metrics.Metrics) *retry {
	return &retry{
		wg:           wg,
		asyncLimiter: asyncLimiter,
		logger:       log.Scope("retry").WithTraceparent(traceparent),
		metrics:      metrics,
		errors:       &Errors{},
	}
}

// Process runs handler under the sink's retry, backoff, and timeout policy.
// For async delivery it releases the WaitGroup immediately and bounds
// concurrency via concurrencyKey; for reliable delivery it releases the
// WaitGroup only after the work completes. Fatal failures on crash-mode
// reliable sinks are recorded and surfaced by Error.
func (r *retry) Process(
	ctx context.Context,
	sink config.SinkConfig,
	concurrencyKey string,
	handler func(ctx context.Context) error,
) {
	sinkType := string(sink.Sink.Type())
	sinkName := sink.Sink.DisplayName()

	if sink.Delivery.Mode == config.DeliveryModeAsync {
		// async: release the processor immediately; work continues in this goroutine
		r.wg.Done()

		// limit concurrency for async sinks
		if err := r.asyncLimiter.Acquire(ctx, concurrencyKey, int64(sink.Delivery.MaxConcurrent)); err != nil {
			return
		}
		defer r.asyncLimiter.Release(concurrencyKey)
	} else {
		// reliable: wait for the processing to complete before releasing the processor
		defer r.wg.Done()
	}

	err := r.runWithRetry(ctx, sinkType, sinkName, sink.Retry, handler)
	if err == nil {
		r.metrics.SinkMessagesSuccess(1, sinkType, sinkName)
	} else {
		if !errors.Is(err, context.Canceled) {
			r.metrics.SinkMessagesFailure(1, sinkType, sinkName)

			r.logger.Error("Processing failed after retries",
				"sink", sinkName,
				"attempts", sink.Retry.MaxAttempts,
				"error", err,
			)

			if sink.Failure.Mode == config.FailureModeCrash && sink.Delivery.Mode != config.DeliveryModeAsync {
				r.errors.Append(fmt.Errorf("%s(%s) processing failed after %d attempts: %w", sinkType, sinkName, sink.Retry.MaxAttempts, err))
			}
		}
	}
}

// Error returns the joined fatal errors recorded by Process across all
// crash-mode reliable sinks, or nil if none occurred.
func (r *retry) Error() error {
	return r.errors.Error()
}

func (r *retry) runWithRetry(
	ctx context.Context,
	sinkType string,
	sinkName string,
	cfg *config.Retry,
	handler func(ctx context.Context) error,
) error {
	maxAttempts := max(cfg.MaxAttempts, 1)

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// skip attempts if context is already done
		if err := ctx.Err(); err != nil {
			return err
		}

		if attempt > 1 {
			r.metrics.SinkMessagesRetry(1, sinkType, sinkName)
		}

		// call the handler with attempt timeout
		start := time.Now()
		err := r.callWithTimeout(ctx, cfg.AttemptTimeout, handler)
		r.metrics.SinkDuration(time.Since(start).Seconds(), sinkType, sinkName)

		if err == nil {
			if attempt > 1 {
				r.logger.Info("Message processed successfully after previous failures",
					"sink", sinkName,
					"attempt", attempt,
				)
			}
			return nil
		}
		lastErr = err

		if errors.Is(err, errs.ErrSinkPermanent) {
			r.logger.Error("Permanent error encountered, will not retry",
				"sink", sinkName,
				"attempt", attempt,
				"maxAttempts", maxAttempts,
				"error", err,
			)
			return err
		}

		if !errors.Is(err, context.Canceled) {
			r.logger.Warn("Processing attempt failed",
				"sink", sinkName,
				"attempt", attempt,
				"maxAttempts", maxAttempts,
				"error", err,
			)
		}

		// exit if we've reached the max attempts
		if attempt == maxAttempts {
			break
		}

		// calculate backoff delay for the next attempt
		delay := r.backoffDelay(cfg.Backoff, attempt)
		if delay <= 0 {
			continue
		}

		// wait for the backoff delay or context cancellation before the next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

func (r *retry) callWithTimeout(ctx context.Context, timeout time.Duration, handler func(ctx context.Context) error) error {
	// setup timeout
	runCtx, cancel := context.WithTimeout(ctx, max(timeout, 0))
	defer cancel()

	// check if context is already done before starting the handler
	if err := runCtx.Err(); err != nil {
		return err
	}

	// run the handler in a separate goroutine
	handlerDone := make(chan error, 1)
	go func() {
		handlerDone <- handler(runCtx)
	}()

	// wait for either the handler to finish or the timeout to expire
	select {
	case <-runCtx.Done():
		return fmt.Errorf("%w: %s", errors.Join(errs.ErrSinkRetryAttemptTimeout, runCtx.Err()), timeout)
	case err := <-handlerDone:
		return err
	}
}

func (r *retry) backoffDelay(b config.Backoff, attempt int) time.Duration {
	var delay time.Duration

	switch b.Type {
	case config.BackoffTypeLinear:
		delay = b.Initial + b.Increment*time.Duration(attempt-1)
	case config.BackoffTypeExponential:
		mult := b.Multiplier
		if mult <= 0 {
			mult = 2
		}

		f := float64(b.Initial)
		for i := 1; i < attempt; i++ {
			f *= mult
		}

		delay = time.Duration(f)
	default:
		panic("Unsupported backoff type")
	}

	if b.Max > 0 && delay > b.Max {
		delay = b.Max
	}

	if b.Jitter == config.JitterModeFull && delay > 0 {
		delay = time.Duration(rand.Int64N(int64(delay)))
	}

	return delay
}

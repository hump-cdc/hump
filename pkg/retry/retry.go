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

	"hump/internal/telemetry/metrics"
	"hump/internal/types/config"
	"hump/internal/types/errs"
	"hump/internal/types/stream"
	"hump/pkg/limit"
	"hump/pkg/telemetry/log"
)

// retry executes sink handlers with retry, backoff, and concurrency control,
// recording metrics and accumulating fatal errors for crash-on-failure sinks.
type retry struct {
	wg      *sync.WaitGroup
	limiter *limit.Limit
	logger  *log.Log
	metrics *metrics.Metrics
	errors  *Errors
}

func New(wg *sync.WaitGroup, traceparent string, limiter *limit.Limit, metrics *metrics.Metrics) *retry {
	return &retry{
		wg:      wg,
		limiter: limiter,
		logger:  log.Scope("retry").WithTraceparent(traceparent),
		metrics: metrics,
		errors:  &Errors{},
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
	host string,
	target string,
	streamEvent stream.Event,
	publisher stream.Publisher,
	handler func(ctx context.Context, evt *stream.Event) error,
) {
	sinkType := string(sink.Sink.Type())
	sinkName := sink.Sink.DisplayName()

	if sink.Delivery.Mode == config.DeliveryModeAsync {
		// async: release the processor immediately; work continues in this goroutine
		r.wg.Done()

		// limit concurrency for async sinks
		if err := r.limiter.Acquire(ctx, concurrencyKey, int64(sink.Delivery.MaxConcurrent)); err != nil {
			return
		}
		defer r.limiter.Release(concurrencyKey)
	} else {
		// reliable: wait for the processing to complete before releasing the processor
		defer r.wg.Done()
	}

	err := r.runWithRetry(ctx, sinkType, sinkName, host, target, sink.Retry, streamEvent, publisher, handler)
	if err == nil {
		r.metrics.SinkMessagesSuccess(1, sinkType, host, target)
	} else {
		if !errors.Is(err, context.Canceled) {
			r.metrics.SinkMessagesFailure(1, sinkType, host, target)

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
	host string,
	target string,
	cfg *config.Retry,
	baseEvent stream.Event,
	publisher stream.Publisher,
	handler func(ctx context.Context, evt *stream.Event) error,
) error {
	maxAttempts := max(cfg.MaxAttempts, 1)

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// skip attempts if context is already done
		if err := ctx.Err(); err != nil {
			return err
		}

		if attempt > 1 {
			r.metrics.SinkMessagesRetry(1, sinkType, host, target)
		}

		// Only attempt 1 fills the operation event; retries publish a lightweight
		// RetryEvent that carries no rendered payload.
		var streamEvent stream.Event
		var evtPtr *stream.Event
		if publisher != nil && attempt == 1 {
			streamEvent = baseEvent
			evtPtr = &streamEvent
		}

		// call the handler with attempt timeout
		start := time.Now()
		err, completed := r.callWithTimeout(ctx, cfg.AttemptTimeout, func(c context.Context) error {
			return handler(c, evtPtr)
		})
		elapsed := time.Since(start)
		r.metrics.SinkDuration(elapsed.Seconds(), sinkType, host, target)

		// Stream the outcome after the publish attempt (success or failure).
		if publisher != nil {
			if attempt == 1 {
				// Read the shared evt only when the handler completed; on timeout
				// its goroutine may still be running, so fall back to our private
				// baseEvent copy to stay race-free.
				out := baseEvent
				if completed {
					out = streamEvent
				}
				out.Success = err == nil
				out.Error = errString(err)
				out.DurationMs = int32(elapsed.Milliseconds())
				publisher.Publish(out)
			} else {
				publisher.Publish(stream.RetryEvent{
					OperationID: baseEvent.ID,
					Attempt:     attempt,
					Success:     err == nil,
					Error:       errString(err),
					Timestamp:   time.Now(),
					DurationMs:  int32(elapsed.Milliseconds()),
				})
			}
		}

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

// callWithTimeout runs handler under an attempt timeout. The returned completed
// flag reports whether the handler actually finished (true) versus the attempt
// timing out while the handler goroutine is still running (false); callers use
// it to know when state the handler mutated is safe to read.
func (r *retry) callWithTimeout(ctx context.Context, timeout time.Duration, handler func(ctx context.Context) error) (error, bool) {
	// setup timeout
	runCtx, cancel := context.WithTimeout(ctx, max(timeout, 0))
	defer cancel()

	// check if context is already done before starting the handler
	if err := runCtx.Err(); err != nil {
		return err, false
	}

	// run the handler in a separate goroutine
	handlerDone := make(chan error, 1)
	go func() {
		handlerDone <- handler(runCtx)
	}()

	// wait for either the handler to finish or the timeout to expire
	select {
	case <-runCtx.Done():
		return fmt.Errorf("%w: %s", errors.Join(errs.ErrSinkRetryAttemptTimeout, runCtx.Err()), timeout), false
	case err := <-handlerDone:
		return err, true
	}
}

// errString returns the error message, or "" when err is nil.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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

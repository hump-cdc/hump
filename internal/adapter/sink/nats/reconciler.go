package nats

import (
	"context"
	"errors"
	"fmt"
	"sync"

	sink "hump/internal/port/sink"

	"github.com/nats-io/nats.go/jetstream"
)

// streamState guards a single stream's reconcile outcome. mu serializes
// concurrent reconcile attempts for the same stream so only one publisher
// runs reconcile at a time. done is set once the reconcile reaches a sticky
// outcome (success or a non-transient failure); err carries that outcome.
// While done is false, the next caller re-runs reconcile — used to recover
// after transient transport errors observed during a cold broker.
type streamState struct {
	mu   sync.Mutex
	done bool
	err  error
}

// reconciler runs the JetStream stream reconcile at most once per stream on
// a single connection, replaying the sticky result to later callers. Transient
// transport failures (broker unreachable, mid-reconnect, context timeout) are
// not cached so the next publish retries reconcile after the connection
// recovers. js is the JetStream context of the owning connection, shared by
// every stream the reconciler reconciles.
type reconciler struct {
	js jetstream.JetStream

	mu     sync.Mutex
	states map[string]*streamState
}

// newReconciler returns a ready-to-use [reconciler] bound to the
// connection's JetStream context.
func newReconciler(js jetstream.JetStream) *reconciler {
	return &reconciler{
		js:     js,
		states: map[string]*streamState{},
	}
}

// stateFor returns the [streamState] guarding reconcile for stream, creating
// one on first use.
func (g *reconciler) stateFor(stream string) *streamState {
	g.mu.Lock()
	defer g.mu.Unlock()

	s, ok := g.states[stream]
	if !ok {
		s = &streamState{}
		g.states[stream] = s
	}

	return s
}

// ensureReconciled runs the JetStream stream reconcile at most once per
// (connection, stream) per process for non-transient outcomes. Returns nil
// when reconcile is not configured. Transient transport errors are returned
// without marking the state done, so the next call retries.
func (g *reconciler) ensureReconciled(ctx context.Context, cfg sink.NatsJetStreamSink) error {
	if len(cfg.Stream.Subjects) == 0 {
		return nil
	}

	state := g.stateFor(cfg.Stream.Name)

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.done {
		return state.err
	}

	err := g.runReconcile(ctx, cfg)
	if isTransientConnError(err) {
		return err
	}

	state.done = true
	state.err = err
	return err
}

// runReconcile builds the [jetstream.StreamConfig] from declared fields and
// dispatches on the reconcile mode. Transient transport errors are returned
// unwrapped so the sink classifier treats them as retriable. API-level
// reconcile failures are wrapped with [errReconcile] so callers can classify
// them as permanent.
func (g *reconciler) runReconcile(ctx context.Context, cfg sink.NatsJetStreamSink) error {
	streamCfg := jetstream.StreamConfig{Name: cfg.Stream.Name}

	if len(cfg.Stream.Subjects) > 0 {
		streamCfg.Subjects = cfg.Stream.Subjects
	}

	if cfg.Retention != "" {
		switch cfg.Retention {
		case "limits":
			streamCfg.Retention = jetstream.LimitsPolicy
		case "interest":
			streamCfg.Retention = jetstream.InterestPolicy
		case "workqueue":
			streamCfg.Retention = jetstream.WorkQueuePolicy
		}
	}

	if cfg.Storage != "" {
		switch cfg.Storage {
		case "file":
			streamCfg.Storage = jetstream.FileStorage
		case "memory":
			streamCfg.Storage = jetstream.MemoryStorage
		}
	}

	var apiErr error
	switch cfg.Mode {
	case sink.JetStreamReconcileCreateWhenMissing:
		_, err := g.js.Stream(ctx, cfg.Stream.Name)
		if err == nil {
			return nil
		}
		if !errors.Is(err, jetstream.ErrStreamNotFound) {
			apiErr = err
			break
		}
		_, apiErr = g.js.CreateStream(ctx, streamCfg)
	default:
		_, apiErr = g.js.CreateOrUpdateStream(ctx, streamCfg)
	}

	if apiErr == nil {
		return nil
	}

	if isTransientConnError(apiErr) {
		return apiErr
	}

	return fmt.Errorf("reconcile stream %s: %w", cfg.Stream.Name, errors.Join(apiErr, errReconcile))
}

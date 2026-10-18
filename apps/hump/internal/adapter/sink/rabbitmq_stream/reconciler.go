package rabbitmq_stream

import (
	"context"
	"errors"
	"sync"

	sinkcfg "hump/config/sink"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

// streamState guards a single stream's reconcile outcome. mu serializes
// concurrent reconcile attempts for the same stream. done is set once the
// reconcile reaches a sticky outcome; transient errors leave done false so
// the next caller retries after the broker recovers.
type streamState struct {
	mu   sync.Mutex
	done bool
	err  error
}

// reconciler runs the stream declaration at most once per stream on a given
// environment for non-transient outcomes, replaying the sticky result.
type reconciler struct {
	mu     sync.Mutex
	states map[string]*streamState
}

func newReconciler() *reconciler {
	return &reconciler{states: map[string]*streamState{}}
}

func (r *reconciler) stateFor(name string) *streamState {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.states[name]
	if !ok {
		s = &streamState{}
		r.states[name] = s
	}
	return s
}

// ensure runs the configured reconcile mode against name. Transient errors
// are returned without marking the state done.
func (r *reconciler) ensure(ctx context.Context, env *stream.Environment, cfg sinkcfg.RabbitMQStreamSink, name string) error {
	state := r.stateFor(name)

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.done {
		return state.err
	}

	err := r.runDeclare(ctx, env, cfg, name)
	if isTransientStreamError(err) {
		return err
	}

	state.done = true
	state.err = err
	return err
}

// runDeclare performs the actual stream declaration. In `passive` mode an
// absent stream surfaces as a permanent [errStreamMissing]. In
// `create-when-missing` the broker silently accepts an already-existing
// declaration (the client swallows StreamAlreadyExists), so the call is
// idempotent. A PreconditionFailed wraps to [errStreamMismatch].
func (r *reconciler) runDeclare(ctx context.Context, env *stream.Environment, cfg sinkcfg.RabbitMQStreamSink, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	switch cfg.Stream.Mode {
	case sinkcfg.RabbitMQStreamModePassive:
		exists, err := env.StreamExists(name)
		if err != nil {
			return err
		}
		if !exists {
			return errStreamMissing
		}
		logger.Info("RabbitMQ stream verified", "stream", name, "mode", string(cfg.Stream.Mode))
		return nil

	default: // create-when-missing
		opts := stream.NewStreamOptions().
			SetMaxAge(cfg.Stream.MaxAge).
			SetMaxLengthBytes(stream.ByteCapacity{}.B(cfg.Stream.MaxLengthBytes)).
			SetMaxSegmentSizeBytes(stream.ByteCapacity{}.B(cfg.Stream.MaxSegmentSizeBytes))

		if err := env.DeclareStream(name, opts); err != nil {
			if errors.Is(err, stream.PreconditionFailed) {
				return errors.Join(errStreamMismatch, err)
			}
			return err
		}
		logger.Info("RabbitMQ stream declared", "stream", name, "mode", string(cfg.Stream.Mode))
		return nil
	}
}

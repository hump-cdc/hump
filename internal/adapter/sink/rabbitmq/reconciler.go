package rabbitmq

import (
	"context"
	"errors"
	"sync"

	sinkcfg "hump/config/sink"

	amqp "github.com/rabbitmq/amqp091-go"
)

// publishChannel is the slice of [amqp.Channel] the reconciler depends on.
// Defined as an interface so tests can supply a fake without dialing.
type publishChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	ExchangeDeclarePassive(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
}

// exchangeState guards a single exchange's reconcile outcome. mu serializes
// concurrent reconcile attempts for the same name so only one publisher
// declares at a time. done is set once the reconcile reaches a sticky
// outcome (success or a non-transient failure); err carries that outcome.
type exchangeState struct {
	mu   sync.Mutex
	done bool
	err  error
}

// exchangeReconciler runs the AMQP exchange declaration at most once per
// (connection, exchange name) for non-transient outcomes, replaying the sticky
// result to later callers. chFn returns the current channel — the function
// indirection keeps the reconciler bound to the post-reconnect channel.
type exchangeReconciler struct {
	chFn func() (*amqp.Channel, error)

	mu     sync.Mutex
	states map[string]*exchangeState
}

// newExchangeReconciler returns an [exchangeReconciler] that pulls the live
// channel via chFn on each declare.
func newExchangeReconciler(chFn func() (*amqp.Channel, error)) *exchangeReconciler {
	return &exchangeReconciler{
		chFn:   chFn,
		states: map[string]*exchangeState{},
	}
}

// stateFor returns the [exchangeState] guarding declarations for exchange,
// creating one on first use.
func (r *exchangeReconciler) stateFor(exchange string) *exchangeState {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.states[exchange]
	if !ok {
		s = &exchangeState{}
		r.states[exchange] = s
	}
	return s
}

// invalidate resets every cached state so the next publish re-declares each
// exchange. Called from the connection supervisor after a reconnect.
func (r *exchangeReconciler) invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = map[string]*exchangeState{}
}

// ensureDeclared runs the configured reconcile mode against name at most once
// per (connection, name) per process for non-transient outcomes. Transient
// transport errors are returned without marking the state done so the next
// call retries after the supervisor reconnects.
func (r *exchangeReconciler) ensureDeclared(ctx context.Context, exchangeCfg sinkcfg.RabbitMQExchange, name string) error {
	state := r.stateFor(name)

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.done {
		return state.err
	}

	err := r.runDeclare(ctx, exchangeCfg, name)
	if isTransientConnError(err) {
		return err
	}

	state.done = true
	state.err = err
	return err
}

// runDeclare performs the actual exchange declaration against the live
// channel. In `passive` mode a 404 is permanent; in `create-when-missing`
// mode a 404 from the passive probe falls through to ExchangeDeclare with
// the configured parameters. A 406 from either path is treated as a
// permanent precondition mismatch.
func (r *exchangeReconciler) runDeclare(ctx context.Context, exchangeCfg sinkcfg.RabbitMQExchange, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ch, err := r.chFn()
	if err != nil {
		return err
	}

	args := amqp.Table(exchangeCfg.Arguments)
	durable := false
	if exchangeCfg.Durable != nil {
		durable = *exchangeCfg.Durable
	}

	switch exchangeCfg.Mode {
	case sinkcfg.RabbitMQExchangeModePassive:
		err := ch.ExchangeDeclarePassive(name, exchangeCfg.Type, durable, exchangeCfg.AutoDelete, false, false, args)
		if err == nil {
			logger.Info("RabbitMQ exchange verified", "exchange", name, "mode", string(exchangeCfg.Mode))
			return nil
		}
		return classifyDeclareError(err)

	default: // create-when-missing
		probeErr := ch.ExchangeDeclarePassive(name, exchangeCfg.Type, durable, exchangeCfg.AutoDelete, false, false, args)
		if probeErr == nil {
			logger.Info("RabbitMQ exchange verified", "exchange", name, "mode", string(exchangeCfg.Mode))
			return nil
		}

		// A passive probe failure closes the channel. The next operation on
		// the same channel returns ErrClosed; surface that as transient so
		// the supervisor reopens before retry. Note: 406 from the passive
		// probe should always be a precondition mismatch, but channel-level
		// errors close the channel either way — defer the actual declare to
		// the next attempt after the channel reopens.
		var amqpErr *amqp.Error
		if errors.As(probeErr, &amqpErr) && amqpErr.Code == amqp.NotFound {
			// Channel is now closed by the broker because of the 404.
			// Fetch a fresh channel and declare with the configured params.
			ch, err = r.chFn()
			if err != nil {
				return err
			}
			declareErr := ch.ExchangeDeclare(name, exchangeCfg.Type, durable, exchangeCfg.AutoDelete, false, false, args)
			if declareErr == nil {
				logger.Info("RabbitMQ exchange declared", "exchange", name, "mode", string(exchangeCfg.Mode))
				return nil
			}
			return classifyDeclareError(declareErr)
		}
		return classifyDeclareError(probeErr)
	}
}

// classifyDeclareError maps a broker error from an exchange declare to one
// of the sink's sentinel errors when the underlying [*amqp.Error] code is a
// known declaration-level failure. Otherwise the original error is returned
// so its transport-level classification (transient vs not) survives.
func classifyDeclareError(err error) error {
	var amqpErr *amqp.Error
	if !errors.As(err, &amqpErr) {
		return err
	}
	switch amqpErr.Code {
	case amqp.NotFound:
		return errors.Join(errExchangeMissing, err)
	case amqp.PreconditionFailed:
		return errors.Join(errExchangeMismatch, err)
	}
	return err
}

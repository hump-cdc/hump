package rabbitmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// returnStragglerWindow bounds how long awaitPublishResult waits for a
// basic.return that the broker may emit *after* the ack of an unroutable
// mandatory publish. The rc.mu invariant (one publish at a time per
// channel) guarantees any straggler in this window belongs to the just-
// completed publish.
const returnStragglerWindow = 50 * time.Millisecond

// drainPending discards any stale confirmation or return left on the channel
// from a previous publish whose caller bailed before reading them. Called
// under rc.mu before each publish so the next read is the new publish's
// result.
func drainPending(confirms chan amqp.Confirmation, returns chan amqp.Return) {
	for {
		select {
		case <-confirms:
		case <-returns:
		default:
			return
		}
	}
}

// awaitPublishResult waits for the publish identified by msg to be
// acknowledged, NACKed, or returned. The function relies on the
// channel-mutex invariant: only one publish is in flight on the channel at
// a time, so the next item on each channel belongs to the just-issued
// publish.
//
// Ordering: when mandatory=true and the message is unroutable, RabbitMQ
// emits the basic.return *before* the basic.ack. Detect that case by
// checking the returns channel first.
func awaitPublishResult(ctx context.Context, rc *rabbitConn, _ *rabbitMessage) error {
	confirms := rc.confirms
	returns := rc.returns

	// Race the three channels. Whichever fires first determines the path.
	select {
	case ret, ok := <-returns:
		if !ok {
			return errReconnecting
		}
		// The broker still emits an ack after returning an unroutable
		// mandatory publish. Drain it under a short window so the next
		// publish sees an empty confirms channel.
		select {
		case <-confirms:
		case <-time.After(returnStragglerWindow):
		}
		_ = ret
		return errUnroutable

	case conf, ok := <-confirms:
		if !ok {
			return errReconnecting
		}
		if !conf.Ack {
			return errPublishNacked
		}
		// Ack with no preceding return — short straggler drain in case a
		// return is in flight for this same publish.
		select {
		case <-returns:
			return errUnroutable
		case <-time.After(returnStragglerWindow):
			return nil
		}

	case <-ctx.Done():
		return ctx.Err()
	}
}

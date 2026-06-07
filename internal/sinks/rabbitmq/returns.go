package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// This file resolves the outcome of a single publish using two broker-driven
// channels wired in client.go:
//
//   - rc.confirms (NotifyPublish): the broker pushes a Confirmation here for
//     every publish once confirm mode is enabled — Ack=true when it accepted
//     the message, Ack=false on a NACK/reject.
//   - rc.returns (NotifyReturn): the broker pushes a Return here ONLY for a
//     mandatory publish that matched no bound queue (unroutable). An
//     unroutable mandatory message is still acked afterwards, so a return is
//     an *extra* signal layered on top of the confirm, not a replacement.
//
// Both functions run under rc.mu, so exactly one publish is in flight on the
// channel at any time. That single-in-flight property, plus drainPending
// clearing leftovers before each publish, is what lets the next item read off
// each channel be attributed unambiguously to the current publish.

// drainPending discards any stale confirmation or return left on the channel
// from a previous publish whose caller bailed before reading them. Called
// under rc.mu before each publish so the next read is the new publish's
// result.
//
// Step by step:
//  1. Loop with a select that has a default arm.
//  2. If a stale Confirmation or Return is buffered, receive and discard it,
//     then loop again to check for more.
//  3. When both channels are empty, default fires and the function returns.
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

// awaitPublishResult waits for the just-issued publish to be acknowledged,
// NACKed, or returned. It relies on two invariants for an exact, zero-latency
// result:
//
//   - Only one publish is in flight on the channel at a time (rc.mu), and
//     drainPending cleared any stale items before this publish — so the next
//     item on each channel belongs to this publish.
//   - amqp091-go dispatches frames from the connection's single reader
//     goroutine in wire order. For an unroutable mandatory publish the broker
//     sends basic.return *before* basic.ack, so the library's send into the
//     returns channel completes before its send into the confirms channel.
//
// Consequently, once this publish's ack is readable, its basic.return (if any)
// is already enqueued on returns — a non-blocking check is exact, with no
// timing window.
//
// Step by step, the outer select races three outcomes:
//
//	Branch 1 — a Return arrives first (unroutable mandatory publish, where the
//	broker emits basic.return then basic.ack):
//	  - A closed channel (ok == false) means the connection died: errReconnecting
//	    (transient, retried upstream).
//	  - Otherwise consume the guaranteed trailing ack so it does not leak into
//	    the next publish's result, bounding the wait on ctx in case the
//	    connection drops before it lands, then return errUnroutable (permanent —
//	    no bound queue, retrying will not help).
//
//	Branch 2 — a Confirmation arrives first (the common, routable path):
//	  - A closed channel means the connection died: errReconnecting.
//	  - Ack == false is a broker NACK: errPublishNacked.
//	  - On an ack, an unroutable mandatory message would also have queued a
//	    return BEFORE this ack, so a non-blocking check of returns is exact:
//	    present -> errUnroutable, absent -> nil immediately. (This default arm
//	    replaced a blind 50ms wait that throttled throughput to ~20 msg/s.)
//
//	Branch 3 — ctx is cancelled before any broker response: return ctx.Err().
func awaitPublishResult(ctx context.Context, rc *rabbitConn, _ *rabbitMessage) error {
	confirms := rc.confirms
	returns := rc.returns

	select {
	// Branch 1: return first — unroutable mandatory publish.
	case _, ok := <-returns:
		if !ok {
			return errReconnecting
		}
		// The trailing ack is guaranteed to follow the return; consume it so it
		// does not leak into the next publish's result.
		select {
		case <-confirms:
		case <-ctx.Done():
		}
		return errUnroutable

	// Branch 2: confirm first — the common routable path (and NACKs).
	case conf, ok := <-confirms:
		if !ok {
			return errReconnecting
		}
		if !conf.Ack {
			return errPublishNacked
		}
		// Any basic.return for this publish is already enqueued by the time its
		// ack arrives — non-blocking check, no straggler window.
		select {
		case <-returns:
			return errUnroutable
		default:
			return nil
		}

	// Branch 3: caller's context expired before the broker responded.
	case <-ctx.Done():
		return ctx.Err()
	}
}

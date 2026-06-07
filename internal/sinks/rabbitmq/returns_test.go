package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// newTestConn wires a rabbitConn with hand-made confirm/return channels so
// awaitPublishResult can be exercised without a broker.
func newTestConn(depth int) *rabbitConn {
	return &rabbitConn{
		confirms: make(chan amqp.Confirmation, depth),
		returns:  make(chan amqp.Return, depth),
	}
}

func TestAwaitPublishResult(t *testing.T) {
	t.Run("ack only returns nil without delay", func(t *testing.T) {
		rc := newTestConn(1)
		rc.confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}

		start := time.Now()
		err := awaitPublishResult(context.Background(), rc, nil)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("want nil, got %v", err)
		}
		// Regression lock for the old 50ms straggler window.
		if elapsed > 5*time.Millisecond {
			t.Fatalf("ack path took %s, want < 5ms (straggler stall regressed)", elapsed)
		}
	})

	t.Run("nack returns errPublishNacked", func(t *testing.T) {
		rc := newTestConn(1)
		rc.confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: false}

		if err := awaitPublishResult(context.Background(), rc, nil); !errors.Is(err, errPublishNacked) {
			t.Fatalf("want errPublishNacked, got %v", err)
		}
	})

	t.Run("return then ack returns errUnroutable and drains the ack", func(t *testing.T) {
		rc := newTestConn(1)
		// Wire order: return is enqueued before the ack.
		rc.returns <- amqp.Return{ReplyCode: amqp.NoRoute}
		// Deliver the trailing ack shortly after, mimicking the broker.
		go func() {
			time.Sleep(time.Millisecond)
			rc.confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}
		}()

		if err := awaitPublishResult(context.Background(), rc, nil); !errors.Is(err, errUnroutable) {
			t.Fatalf("want errUnroutable, got %v", err)
		}
		// Trailing ack must have been consumed.
		select {
		case c := <-rc.confirms:
			t.Fatalf("trailing ack leaked: %+v", c)
		default:
		}
	})

	t.Run("ack picked first with return also enqueued returns errUnroutable", func(t *testing.T) {
		rc := newTestConn(1)
		// Both already buffered: the top-level select may pick either case.
		// The ack branch must still detect the pending return.
		rc.returns <- amqp.Return{ReplyCode: amqp.NoRoute}
		rc.confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}

		if err := awaitPublishResult(context.Background(), rc, nil); !errors.Is(err, errUnroutable) {
			t.Fatalf("want errUnroutable, got %v", err)
		}
	})

	t.Run("closed confirms returns errReconnecting", func(t *testing.T) {
		rc := newTestConn(1)
		close(rc.confirms)

		if err := awaitPublishResult(context.Background(), rc, nil); !errors.Is(err, errReconnecting) {
			t.Fatalf("want errReconnecting, got %v", err)
		}
	})

	t.Run("closed returns yields errReconnecting", func(t *testing.T) {
		rc := newTestConn(1)
		close(rc.returns)

		if err := awaitPublishResult(context.Background(), rc, nil); !errors.Is(err, errReconnecting) {
			t.Fatalf("want errReconnecting, got %v", err)
		}
	})

	t.Run("cancelled context with nothing pending returns ctx error", func(t *testing.T) {
		rc := newTestConn(1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := awaitPublishResult(ctx, rc, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	})
}

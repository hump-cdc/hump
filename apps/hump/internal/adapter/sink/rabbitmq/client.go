package rabbitmq

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/observability/diagnostic"
	"hump/pkg/version"
)

// AMQP connection tuning shared by every cached connection.
const (
	dialTimeout       = 10 * time.Second
	heartbeatInterval = 10 * time.Second

	reconnectBase       = 2 * time.Second
	minReconnectJitter  = 100 * time.Millisecond
	maxReconnectJitter  = 1 * time.Second
	confirmChannelDepth = 1
)

// clientKey identifies a cached AMQP connection by its rendered URL. Auth
// lives inside the URL string, so the URL alone keys the cache.
type clientKey struct {
	url string
}

// rabbitConn holds the connection, confirm-mode channel and exchange reconciler.
// mu serializes publishing and confirmation drains; see returns.go for the invariant.
type rabbitConn struct {
	url string

	mu         sync.Mutex
	conn       *amqp.Connection
	ch         *amqp.Channel
	confirms   chan amqp.Confirmation
	returns    chan amqp.Return
	closeNotif chan *amqp.Error

	reconciler *exchangeReconciler

	readyMu sync.RWMutex
	ready   chan struct{}

	quit chan struct{}
}

var (
	clientsMu sync.Mutex
	clients   = map[clientKey]*rabbitConn{}
)

// getClient returns a cached [rabbitConn] for the rendered URL described by
// cfg, dialing on first use and registering a [shutdown.OnShutdown] hook so
// the connection is drained at process exit.
func getClient(cfg sinkcfg.RabbitMQSink, operationContext *sink.Context) (*rabbitConn, error) {
	url, err := cfg.URL.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render url: %w", err)
	}

	key := clientKey{url: url}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if c, ok := clients[key]; ok {
		return c, nil
	}

	rc := &rabbitConn{
		url:   url,
		ready: make(chan struct{}),
		quit:  make(chan struct{}),
	}
	rc.reconciler = newExchangeReconciler(rc.healthyChannel)

	if err := rc.dialAndOpen(); err != nil {
		return nil, diagnostic.Error("rabbitmq dial "+diagnostic.Endpoint(url), err)
	}

	go rc.supervise()

	clients[key] = rc

	shutdown.OnShutdown(shutdown.StageSinks, fmt.Sprintf("rabbitmq_sink(%s)", diagnostic.Endpoint(url)), func(_ context.Context) error {
		// Signal the supervisor to stop, then close the channel and connection.
		// Idempotency: closing an already-closed channel panics, guard with
		// a select that races against the supervisor.
		select {
		case <-rc.quit:
		default:
			close(rc.quit)
		}

		rc.mu.Lock()
		ch := rc.ch
		conn := rc.conn
		rc.ch = nil
		rc.conn = nil
		rc.mu.Unlock()

		if ch != nil {
			_ = ch.Close()
		}
		if conn != nil && !conn.IsClosed() {
			_ = conn.Close()
		}

		clientsMu.Lock()
		delete(clients, key)
		clientsMu.Unlock()

		return nil
	})

	return rc, nil
}

// dialAndOpen establishes a connection and confirm-mode channel, then signals readiness.
// The constructor creates rc.ready; the supervisor replaces it before redialing.
func (rc *rabbitConn) dialAndOpen() error {
	conn, err := amqp.DialConfig(rc.url, amqp.Config{
		Heartbeat: heartbeatInterval,
		Dial:      amqp.DefaultDial(dialTimeout),
		Properties: amqp.Table{
			"connection_name": "hump",
			"product":         "hump",
			"version":         version.Version,
		},
	})
	if err != nil {
		return diagnostic.Error("RabbitMQ connection failed", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("enable confirms: %w", err)
	}

	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, confirmChannelDepth))
	returns := ch.NotifyReturn(make(chan amqp.Return, confirmChannelDepth))
	closeNotif := conn.NotifyClose(make(chan *amqp.Error, 1))

	rc.mu.Lock()
	rc.conn = conn
	rc.ch = ch
	rc.confirms = confirms
	rc.returns = returns
	rc.closeNotif = closeNotif
	rc.mu.Unlock()

	rc.signalReady()

	logger.Info("RabbitMQ connection established", "url", diagnostic.Endpoint(rc.url))

	return nil
}

// signalReady closes the current ready channel so blocked callers proceed.
// Replacing the channel happens in invalidateReady after a disconnect.
func (rc *rabbitConn) signalReady() {
	rc.readyMu.Lock()
	defer rc.readyMu.Unlock()
	select {
	case <-rc.ready:
		// already closed
	default:
		close(rc.ready)
	}
}

// invalidateReady replaces ready with a fresh open channel so callers block
// again until the next signalReady.
func (rc *rabbitConn) invalidateReady() {
	rc.readyMu.Lock()
	defer rc.readyMu.Unlock()
	rc.ready = make(chan struct{})
}

// readyChan returns the current ready channel under read lock so callers can
// safely select on it across reconnect cycles.
func (rc *rabbitConn) readyChan() chan struct{} {
	rc.readyMu.RLock()
	defer rc.readyMu.RUnlock()
	return rc.ready
}

// healthyChannel returns the cached channel, transparently reopening it if
// a prior channel-level error (e.g. 404 NOT_FOUND from a passive declare)
// closed it without taking the connection down. Used by the reconciler so
// it always sees a usable channel.
func (rc *rabbitConn) healthyChannel() (*amqp.Channel, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.healthyChannelLocked()
}

// healthyChannelLocked is healthyChannel for callers that already hold rc.mu
// (e.g. the publish path which must hold the mutex across PublishWithContext
// and the subsequent confirm/return drain).
func (rc *rabbitConn) healthyChannelLocked() (*amqp.Channel, error) {
	if rc.conn == nil || rc.conn.IsClosed() {
		return nil, errReconnecting
	}
	if rc.ch != nil && !rc.ch.IsClosed() {
		return rc.ch, nil
	}
	return rc.reopenChannelLocked()
}

// reopenChannelLocked replaces the dead channel in confirm mode; callers must hold rc.mu.
// Exchange declarations survive channel replacement, so the reconciler stays valid.
func (rc *rabbitConn) reopenChannelLocked() (*amqp.Channel, error) {
	ch, err := rc.conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("enable confirms: %w", err)
	}

	rc.ch = ch
	rc.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, confirmChannelDepth))
	rc.returns = ch.NotifyReturn(make(chan amqp.Return, confirmChannelDepth))

	logger.Info("RabbitMQ channel reopened", "url", diagnostic.Endpoint(rc.url))
	return ch, nil
}

// supervise watches for connection-close events and re-establishes the
// connection with infinite 2s + jitter backoff. Exits cleanly when quit is
// closed by the shutdown hook.
func (rc *rabbitConn) supervise() {
	for {
		rc.mu.Lock()
		closeNotif := rc.closeNotif
		rc.mu.Unlock()

		var amqpErr *amqp.Error
		select {
		case <-rc.quit:
			return
		case e, ok := <-closeNotif:
			if !ok {
				// channel closed without an error frame — happens on shutdown.
				return
			}
			amqpErr = e
		}

		// Connection died. Surface the cause and replace ready so subsequent
		// Process calls block (until ctx) instead of operating on a dead channel.
		logger.Warn("RabbitMQ connection closed", "url", diagnostic.Endpoint(rc.url), "err", diagnostic.Error("RabbitMQ connection closed", amqpErr))

		rc.mu.Lock()
		rc.ch = nil
		rc.conn = nil
		rc.mu.Unlock()
		rc.invalidateReady()

		// Reconnect with backoff. Aborts on shutdown.
		for {
			wait := reconnectBase + jitter(minReconnectJitter, maxReconnectJitter)
			select {
			case <-rc.quit:
				return
			case <-time.After(wait):
			}

			if err := rc.dialAndOpen(); err != nil {
				logger.Warn("RabbitMQ reconnect failed", "url", diagnostic.Endpoint(rc.url), "err", diagnostic.Error("RabbitMQ reconnect failed", err))
				continue
			}

			logger.Info("RabbitMQ reconnected", "url", diagnostic.Endpoint(rc.url))
			rc.reconciler.invalidate()
			break
		}
	}
}

// jitter returns a duration uniformly distributed in [minD, maxD]. Adds
// per-host scheduling variance so a herd of reconnects does not pulse the
// broker on the same tick.
func jitter(minD, maxD time.Duration) time.Duration {
	if maxD <= minD {
		return minD
	}
	delta := maxD - minD
	return minD + time.Duration(rand.Int64N(int64(delta))) //nolint:gosec // reconnect jitter, not security-sensitive
}

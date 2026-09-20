package operationscollector

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	port "hump/internal/port/operations_collector"
	"hump/pkg/app/shutdown"
)

const (
	// defaultCapacity is how many deliveries the ring keeps.
	defaultCapacity = 1000
	// defaultPayloadLimit caps the payload bytes one record retains.
	defaultPayloadLimit = 64 * 1024
	// defaultIdleTimeout is how long the ring outlives the last client, so a
	// page reload does not throw the log away.
	defaultIdleTimeout = 5 * time.Minute
)

// Collector implements [port.Collector]. It holds no buffer at all until the
// first client calls Connect, and drops the whole ring once no client has been
// connected for its idle timeout.
type Collector struct {
	capacity     int
	payloadLimit int
	idleTimeout  time.Duration
	active       atomic.Bool // active mirrors whether ring is set. Connected reads it on every delivery and must not take mu to do so.
	mu           sync.Mutex
	ring         *ring
	connections  int
	generation   uint64 // generation changes on every Connect, so an idle timer armed before it does not deactivate the collector.
	lastSeq      uint64 // lastSeq carries the record numbering across an idle period, so a reconnecting reader cannot see a sequence it already saw.
	idle         *time.Timer
	closed       bool
}

// Option configures a Collector.
type Option func(*Collector)

// WithCapacity sets how many of the newest deliveries the ring keeps. A
// non-positive n leaves the default.
func WithCapacity(n int) Option {
	return func(c *Collector) {
		if n > 0 {
			c.capacity = n
		}
	}
}

// WithPayloadLimit caps the payload bytes one record retains; a longer body is
// truncated and its original size reported on the record. A non-positive n
// leaves the default.
func WithPayloadLimit(n int) Option {
	return func(c *Collector) {
		if n > 0 {
			c.payloadLimit = n
		}
	}
}

// WithIdleTimeout sets how long the ring outlives the last client.
func WithIdleTimeout(d time.Duration) Option {
	return func(c *Collector) {
		if d > 0 {
			c.idleTimeout = d
		}
	}
}

// New returns an idle collector and registers its shutdown hook.
func New(opts ...Option) *Collector {
	c := &Collector{
		capacity:     defaultCapacity,
		payloadLimit: defaultPayloadLimit,
		idleTimeout:  defaultIdleTimeout,
	}

	for _, opt := range opts {
		opt(c)
	}

	shutdown.OnShutdown(shutdown.StageWorkers, "operations_collector", c.onShutdown)

	return c
}

// Connect registers a client, cancels a pending deactivation and allocates the
// ring if the collector is idle.
func (c *Collector) Connect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.connections++
	c.generation++

	if c.idle != nil {
		c.idle.Stop()
		c.idle = nil
	}

	if c.ring == nil {
		c.activate()
	}
}

// Disconnect unregisters a client and, once none is left, arms the idle timer.
func (c *Collector) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.connections == 0 {
		return
	}

	c.connections--
	if c.connections > 0 {
		return
	}

	generation := c.generation
	c.idle = time.AfterFunc(c.idleTimeout, func() { c.deactivate(generation) })
}

// Connected reports whether the collector is recording. It is a hint rather
// than a guarantee: the last client may leave before the Record that follows,
// which Record handles by doing nothing.
func (c *Collector) Connected() bool {
	return c.active.Load()
}

// Record stores rec, swapping its buffers for reusable ones. It does nothing
// while the collector is idle.
func (c *Collector) Record(rec *port.Record) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ring == nil {
		return
	}

	c.ring.write(rec)
}

// Dump returns the records in [start, end) counted from the newest. An idle
// collector answers an empty page.
func (c *Collector) Dump(start, end int) port.Page {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ring == nil {
		return port.Page{}
	}

	return c.ring.page(start, end)
}

// activate allocates the ring and starts recording. c.mu must be held.
func (c *Collector) activate() {
	c.ring = newRing(c.capacity, c.payloadLimit, c.lastSeq)
	c.active.Store(true)

	logger.Info("Operations collector started", "capacity", c.capacity, "payloadLimit", c.payloadLimit)
}

// deactivate drops the ring unless a client connected after the idle timer of
// generation was armed.
func (c *Collector) deactivate(generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.generation != generation || c.connections > 0 {
		return
	}

	c.stop()
}

// stop drops the ring and the idle timer, returning every buffer to the
// garbage collector. Nothing survives it: an idle collector must cost nothing.
// c.mu must be held.
func (c *Collector) stop() {
	if c.ring != nil {
		c.lastSeq = c.ring.seq
		c.ring = nil
		c.active.Store(false)
		logger.Info("Operations collector stopped")
	}

	if c.idle != nil {
		c.idle.Stop()
		c.idle = nil
	}
}

// onShutdown implements the shutdown hook registered by New.
func (c *Collector) onShutdown(context.Context) error {
	c.close()

	return nil
}

// close stops the collector for good; a later Connect does nothing.
func (c *Collector) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	c.stop()
}

// Package operationscollector is the dashboard's operations log: a bounded ring of the
// newest deliveries, one record per row per sink, allocated only while a client is connected.
package operationscollector

import (
	"context"
	"sync"
	"time"

	port "hump/internal/port/operations_collector"
	"hump/pkg/app/shutdown"
	idlegate "hump/pkg/idle_gate"
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
	gate         *idlegate.Gate
	mu           sync.Mutex
	ring         *ring
	lastSeq      uint64 // lastSeq carries the record numbering across an idle period, so a reconnecting reader cannot see a sequence it already saw.
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

	c.gate = idlegate.New(c.idleTimeout, c.activate, c.stop)

	shutdown.OnShutdown(shutdown.StageWorkers, "operations_collector", c.onShutdown)

	return c
}

// Connect registers a client, cancels a pending deactivation and allocates the
// ring if the collector is idle.
func (c *Collector) Connect() {
	c.gate.Connect()
}

// Disconnect unregisters a client and, once none is left, arms the idle timer.
func (c *Collector) Disconnect() {
	c.gate.Disconnect()
}

// Connected reports whether the collector is recording. It is a hint rather
// than a guarantee: the last client may leave before the Record that follows,
// which Record handles by doing nothing.
func (c *Collector) Connected() bool {
	return c.gate.Active()
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

// Search returns the window of q cut out of the records matching it. An idle
// collector answers an empty page.
func (c *Collector) Search(q port.Query) port.Page {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ring == nil {
		return port.Page{}
	}

	return c.ring.search(q)
}

// activate allocates the ring and starts recording. The gate calls it while
// holding its own lock, so it must not call back into the gate.
func (c *Collector) activate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ring = newRing(c.capacity, c.payloadLimit, c.lastSeq)

	logger.Info("Operations collector started", "capacity", c.capacity, "payloadLimit", c.payloadLimit)
}

// stop drops the ring, returning every buffer to the garbage collector. Only
// the sequence survives it: an idle collector must cost nothing. The gate calls
// it while holding its own lock, so it must not call back into the gate.
func (c *Collector) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSeq = c.ring.seq
	c.ring = nil

	logger.Info("Operations collector stopped")
}

// onShutdown implements the shutdown hook registered by New.
func (c *Collector) onShutdown(context.Context) error {
	c.close()

	return nil
}

// close stops the collector for good; a later Connect does nothing.
func (c *Collector) close() {
	c.gate.Close()
}

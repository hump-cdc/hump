// Package metricscollector scrapes the process metrics registry on demand and
// aggregates the dashboard metrics into 1m/5m/10m buckets kept in memory.
package metricscollector

import (
	"context"
	"sync"
	"time"

	"hump/internal/domain"
	port "hump/internal/port/metrics_collector"
	"hump/pkg/app/shutdown"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// defaultInterval is how often an active collector scrapes the registry.
	defaultInterval = 10 * time.Second
	// defaultIdleTimeout is how long the collector stays active after the last
	// client disconnects.
	defaultIdleTimeout = 5 * time.Minute
	// defaultDebounce is the minimum gap between a scrape and one requested by Scrape.
	defaultDebounce = time.Second
	// pointsMax caps the buckets kept per duration.
	pointsMax = 20
)

// Collector implements [port.Collector]. It stays idle until the first client
// connects, and returns to idle, dropping everything it aggregated, once no
// client has been connected for its idle timeout.
type Collector struct {
	gatherer    prometheus.Gatherer
	interval    time.Duration
	idleTimeout time.Duration
	debounce    time.Duration
	topology    []port.Connection // topology is the configured set of edges the flat views are keyed by; it never changes.
	// recordedMu guards recorded on its own, keeping Record off the lock Dump
	// holds while it merges buckets.
	recordedMu  sync.Mutex
	recorded    map[port.Connection]map[domain.DeliveryStatus]float64
	mu          sync.Mutex
	connections int
	generation  uint64 // generation changes on every Connect, so an idle timer armed before it does not deactivate the collector.
	idle        *time.Timer
	cancel      context.CancelFunc // cancel stops the scrape loop; it is nil while the collector is idle.
	requests    chan struct{}      // requests carries Scrape calls to the scrape loop; it is nil while the collector is idle.
	closed      bool
	scraper     *scraper
	series      map[port.Duration]*series
	lastTick    time.Time
}

// Option configures a [Collector].
type Option func(*Collector)

// WithInterval sets how often an active collector scrapes the registry.
func WithInterval(interval time.Duration) Option {
	return func(c *Collector) { c.interval = interval }
}

// WithIdleTimeout sets how long the collector stays active after the last
// client disconnects.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *Collector) { c.idleTimeout = timeout }
}

// WithDebounce sets the minimum gap between a scrape and one requested by Scrape.
func WithDebounce(debounce time.Duration) Option {
	return func(c *Collector) { c.debounce = debounce }
}

// WithTopology sets the configured table-to-sink-target edges a snapshot reports,
// so an edge with no traffic is still listed.
func WithTopology(conns []port.Connection) Option {
	return func(c *Collector) { c.topology = conns }
}

// New returns an idle collector over gatherer and registers a shutdown hook
// that stops it.
func New(gatherer prometheus.Gatherer, opts ...Option) *Collector {
	c := &Collector{
		gatherer:    gatherer,
		interval:    defaultInterval,
		idleTimeout: defaultIdleTimeout,
		debounce:    defaultDebounce,
		recorded:    make(map[port.Connection]map[domain.DeliveryStatus]float64),
	}

	for _, opt := range opts {
		opt(c)
	}

	shutdown.OnShutdown(shutdown.StageWorkers, "metrics_collector", c.onShutdown)

	return c
}

// onShutdown is the shutdown hook; it stops the collector for good.
func (c *Collector) onShutdown(_ context.Context) error {
	c.close()
	return nil
}

// Connect registers a client, cancels a pending deactivation and starts the
// scrape loop if the collector is idle.
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

	if c.cancel == nil {
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
	c.idle = time.AfterFunc(c.idleTimeout,
		func() {
			c.deactivate(generation)
		})
}

// Scrape asks the scrape loop for an extra scrape. It does nothing while the
// collector is idle, and never blocks.
func (c *Collector) Scrape() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.requests == nil {
		return
	}

	select {
	case c.requests <- struct{}{}:
	default:
	}
}

// Record counts n deliveries over conn that ended with status. Unlike the
// scraped counts it survives an idle period, so it stays comparable with the
// process-lifetime counters it is reported next to.
func (c *Collector) Record(conn port.Connection, status domain.DeliveryStatus, n int) {
	c.recordedMu.Lock()
	defer c.recordedMu.Unlock()

	if c.recorded == nil {
		return
	}

	byStatus, ok := c.recorded[conn]
	if !ok {
		byStatus = make(map[domain.DeliveryStatus]float64, 3)
		c.recorded[conn] = byStatus
	}

	byStatus[status] += float64(n)
}

// Dump returns the newest snapshot aggregated by d.
func (c *Collector) Dump(d port.Duration) (port.Snapshot, error) {
	if _, err := d.Duration(); err != nil {
		return port.Snapshot{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.snapshot(d), nil
}

// activate resets the aggregates and starts the scrape loop. c.mu must be held.
func (c *Collector) activate() {
	ctx, cancel := context.WithCancel(context.Background())

	logger.Info("Metrics collector started", "interval", c.interval)

	c.cancel = cancel
	c.requests = make(chan struct{}, 1)
	c.scraper = newScraper()
	c.series = make(map[port.Duration]*series, len(port.Durations))
	c.lastTick = time.Time{}

	for _, d := range port.Durations {
		duration, _ := d.Duration()
		c.series[d] = &series{duration: duration}
	}

	go c.run(ctx, c.requests)
}

// deactivate stops the collector unless a client connected after the idle
// timer of generation was armed.
func (c *Collector) deactivate(generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.generation != generation || c.connections > 0 {
		return
	}

	c.stop()
}

// close stops the collector for good. It is idempotent.
func (c *Collector) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	c.stop()

	c.recordedMu.Lock()
	c.recorded = nil
	c.recordedMu.Unlock()
}

// stop cancels the scrape loop and the idle timer and drops every aggregate.
// c.mu must be held.
func (c *Collector) stop() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil

		logger.Info("Metrics collector stopped")
	}

	if c.idle != nil {
		c.idle.Stop()
		c.idle = nil
	}

	c.requests = nil
	c.scraper = nil
	c.series = nil
	c.lastTick = time.Time{}
}

// run scrapes once right away, then every interval and on requests until ctx is
// cancelled. A request within debounce of the last scrape is deferred to the end
// of the window, and requests arriving while one is deferred collapse into it.
func (c *Collector) run(ctx context.Context, requests <-chan struct{}) {
	c.tick(ctx)
	last := time.Now()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	var deferred *time.Timer
	var deferredC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// A scheduled scrape also serves a deferred request.
			if deferred != nil {
				deferred.Stop()
				deferred, deferredC = nil, nil
			}

			c.tick(ctx)
			last = time.Now()

		case <-requests:
			if deferred != nil {
				continue
			}

			if wait := c.debounce - time.Since(last); wait > 0 {
				deferred = time.NewTimer(wait)
				deferredC = deferred.C
				continue
			}

			c.tick(ctx)
			last = time.Now()

		case <-deferredC:
			deferred, deferredC = nil, nil

			c.tick(ctx)
			last = time.Now()
		}
	}
}

// tick gathers the registry and folds the result into every series. The first
// tick after activation only records baselines and gauges.
func (c *Collector) tick(ctx context.Context) {
	families, err := c.gatherer.Gather()
	if err != nil {
		logger.Warn("Unable to gather metrics", "error", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// The collector may have been deactivated while gathering; its aggregates
	// are gone and must stay empty.
	if ctx.Err() != nil {
		return
	}

	now := time.Now()

	var elapsed time.Duration
	if !c.lastTick.IsZero() {
		elapsed = now.Sub(c.lastTick)
	}
	c.lastTick = now

	observed := c.scraper.scrape(families)
	for _, s := range c.series {
		s.add(now, elapsed, observed)
	}
}

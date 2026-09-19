package metricscollector

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	port "hump/internal/port/metrics_collector"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestScraperRecordsBaselineOnFirstSighting(t *testing.T) {
	m := newTestMetrics()
	s := newScraper()

	m.messages.WithLabelValues("success", "kafka", "h", "t").Add(5)
	first := s.scrape(m.gather(t))
	if first.delivered != 0 || len(first.sinkDelivery) != 0 {
		t.Fatalf("got %+v on first scrape, want no deltas", first)
	}

	m.messages.WithLabelValues("success", "kafka", "h", "t").Add(3)
	m.messages.WithLabelValues("failure", "kafka", "h", "t").Add(1)
	m.messages.WithLabelValues("retry", "kafka", "h", "t").Add(7)
	m.messages.WithLabelValues("success", "http", "h", "t").Add(2)
	second := s.scrape(m.gather(t))

	if second.delivered != 3 || second.failed != 0 {
		t.Fatalf("got delivered %v failed %v, want 3 and 0", second.delivered, second.failed)
	}

	if second.sinkDelivery["kafka"] != 3 {
		t.Fatalf("got sink delivery %v", second.sinkDelivery)
	}

	third := s.scrape(m.gather(t))
	if third.failed != 0 || third.delivered != 0 || third.sinkDelivery["kafka"] != 0 {
		t.Fatalf("got %+v, want no advance", third)
	}
}

func TestScraperReadsGaugesAndLatency(t *testing.T) {
	m := newTestMetrics()
	s := newScraper()

	m.duration.WithLabelValues("kafka", "a", "t").Observe(0.001)
	s.scrape(m.gather(t))

	m.lag.Set(42)
	m.memory.Set(1024)
	for range 10 {
		m.duration.WithLabelValues("kafka", "a", "t").Observe(0.2)
		m.duration.WithLabelValues("kafka", "b", "t").Observe(0.2)
	}
	observed := s.scrape(m.gather(t))

	if observed.lag != 42 || observed.memory != 1024 {
		t.Fatalf("got lag %v memory %v", observed.lag, observed.memory)
	}

	// Series "b" was first seen on this scrape, so only "a" contributes.
	latency := observed.sinkLatency[sinkKey{sinkType: "kafka", host: "a", target: "t"}]
	if latency == nil || latency.count != 10 {
		t.Fatalf("got latency %+v, want 10 observations", latency)
	}
}

func TestCollectorIdleUntilConnect(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour))
	t.Cleanup(c.close)

	time.Sleep(30 * time.Millisecond)

	if g.calls.Load() != 0 {
		t.Fatalf("gathered %d times while idle", g.calls.Load())
	}

	snapshot, err := c.Dump(port.Duration1m)
	if err != nil || len(snapshot.Points) != 0 {
		t.Fatalf("got %v, %v, want no points", snapshot.Points, err)
	}
}

func TestCollectorCollectsWhileConnected(t *testing.T) {
	m := newTestMetrics()
	c := New(&countingGatherer{metrics: m}, WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour))
	t.Cleanup(c.close)

	m.messages.WithLabelValues("success", "kafka", "h", "t").Add(0)
	c.Connect()

	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration10m)) > 0 })
	m.messages.WithLabelValues("success", "kafka", "h", "t").Add(4)

	waitFor(t, func() bool {
		total := 0.0
		for _, p := range mustDump(t, c, port.Duration10m) {
			total += p.Delivered
		}
		return total == 4
	})
}

func TestCollectorClearsAfterIdleTimeout(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(5*time.Millisecond), WithIdleTimeout(30*time.Millisecond))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })
	c.Disconnect()

	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) == 0 })

	calls := g.calls.Load()
	time.Sleep(30 * time.Millisecond)
	if g.calls.Load() != calls {
		t.Fatal("kept gathering after deactivation")
	}
}

func TestCollectorReconnectCancelsIdleTimeout(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()}, WithInterval(5*time.Millisecond), WithIdleTimeout(30*time.Millisecond))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })
	c.Disconnect()
	c.Connect()

	time.Sleep(80 * time.Millisecond)

	if len(mustDump(t, c, port.Duration1m)) == 0 {
		t.Fatal("deactivated while a client was connected")
	}
}

func TestCollectorIgnoresConnectAfterClose(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour))

	c.close()
	c.Connect()
	c.Disconnect()
	time.Sleep(30 * time.Millisecond)

	if g.calls.Load() != 0 {
		t.Fatalf("gathered %d times after close", g.calls.Load())
	}
}

func TestCollectorRejectsUnknownDuration(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()})
	t.Cleanup(c.close)

	if _, err := c.Dump("2m"); err != port.ErrUnknownDuration {
		t.Fatalf("got %v, want ErrUnknownDuration", err)
	}
}

func TestScraperSkipsSeriesWithoutStatus(t *testing.T) {
	registry := prometheus.NewRegistry()
	counter := prometheus.NewCounter(prometheus.CounterOpts{Name: familySinkMessages})
	registry.MustRegister(counter)
	s := newScraper()

	for range 2 {
		counter.Add(1)

		families, err := registry.Gather()
		if err != nil {
			t.Fatal(err)
		}

		observed := s.scrape(families)
		if observed.delivered != 0 || observed.failed != 0 || len(observed.sinkDelivery) != 0 {
			t.Fatalf("got %+v, want nothing counted", observed)
		}
	}
}

func TestCollectorStaysActiveWhileClientsRemain(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()}, WithInterval(5*time.Millisecond), WithIdleTimeout(10*time.Millisecond))
	t.Cleanup(c.close)

	c.Connect()
	c.Connect()
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })
	c.Disconnect()

	time.Sleep(50 * time.Millisecond)

	if len(mustDump(t, c, port.Duration1m)) == 0 {
		t.Fatal("deactivated while a client was connected")
	}
}

func TestCollectorIgnoresStaleDeactivation(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()}, WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })
	c.deactivate(0)

	if len(mustDump(t, c, port.Duration1m)) == 0 {
		t.Fatal("deactivated by a stale idle timer")
	}
}

func TestCollectorSkipsFailedGather(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics(), err: errors.New("gather failed")}
	c := New(g, WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return g.calls.Load() > 1 })

	if points := mustDump(t, c, port.Duration1m); len(points) != 0 {
		t.Fatalf("got %d points from failed gathers, want none", len(points))
	}
}

func TestCollectorDropsTickAfterCancel(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()})
	t.Cleanup(c.close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.tick(ctx)

	if points := mustDump(t, c, port.Duration1m); len(points) != 0 {
		t.Fatalf("got %d points from a cancelled tick, want none", len(points))
	}
}

func TestCollectorStopsOnShutdown(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour))

	c.Connect()
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })

	if err := c.onShutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	c.Connect()
	if points := mustDump(t, c, port.Duration1m); len(points) != 0 {
		t.Fatalf("got %d points after shutdown, want none", len(points))
	}
}

func TestCollectorScrapeIgnoredWhileIdle(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(time.Hour), WithIdleTimeout(time.Hour), WithDebounce(0))
	t.Cleanup(c.close)

	c.Scrape()
	time.Sleep(30 * time.Millisecond)

	if g.calls.Load() != 0 {
		t.Fatalf("gathered %d times while idle", g.calls.Load())
	}
}

func TestCollectorScrapeTicksOutsideDebounce(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(time.Hour), WithIdleTimeout(time.Hour), WithDebounce(10*time.Millisecond))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return g.calls.Load() == 1 })
	time.Sleep(20 * time.Millisecond)

	c.Scrape()
	waitFor(t, func() bool { return g.calls.Load() == 2 })
}

func TestCollectorScrapeDefersBurstWithinDebounce(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(time.Hour), WithIdleTimeout(time.Hour), WithDebounce(100*time.Millisecond))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return g.calls.Load() == 1 })

	for range 5 {
		c.Scrape()
	}

	time.Sleep(20 * time.Millisecond)
	if g.calls.Load() != 1 {
		t.Fatalf("gathered %d times within debounce, want 1", g.calls.Load())
	}

	waitFor(t, func() bool { return g.calls.Load() == 2 })
	time.Sleep(150 * time.Millisecond)

	if g.calls.Load() != 2 {
		t.Fatalf("gathered %d times after the burst, want 2", g.calls.Load())
	}
}

func TestCollectorTickServesDeferredScrape(t *testing.T) {
	g := &countingGatherer{metrics: newTestMetrics()}
	c := New(g, WithInterval(200*time.Millisecond), WithIdleTimeout(time.Hour), WithDebounce(300*time.Millisecond))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return g.calls.Load() == 1 })
	c.Scrape()

	// The scheduled tick at 200ms serves the request deferred to 300ms, so nothing
	// ticks between them and the next scheduled tick at 400ms.
	time.Sleep(350 * time.Millisecond)

	if g.calls.Load() != 2 {
		t.Fatalf("gathered %d times, want 2", g.calls.Load())
	}
}

// testMetrics is a registry carrying the families the collector reads.
type testMetrics struct {
	registry *prometheus.Registry
	messages *prometheus.CounterVec
	duration *prometheus.HistogramVec
	rows     *prometheus.HistogramVec
	lag      prometheus.Gauge
	memory   prometheus.Gauge
}

// newTestMetrics registers the collector's families on a fresh registry.
func newTestMetrics() *testMetrics {
	m := &testMetrics{
		registry: prometheus.NewRegistry(),
		messages: prometheus.NewCounterVec(prometheus.CounterOpts{Name: familySinkMessages}, []string{"status", "type", "host", "target"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: familySinkDuration}, []string{"type", "host", "target"}),
		rows:     prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: familyProcessedRows}, []string{"operation", "schema", "table"}),
		lag:      prometheus.NewGauge(prometheus.GaugeOpts{Name: familyReplicationLag}),
		memory:   prometheus.NewGauge(prometheus.GaugeOpts{Name: familyMemoryAlloc}),
	}
	m.registry.MustRegister(m.messages, m.duration, m.rows, m.lag, m.memory)

	return m
}

// gather gathers the registry, failing t on error.
func (m *testMetrics) gather(t *testing.T) []*dto.MetricFamily {
	t.Helper()

	families, err := m.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	return families
}

// countingGatherer gathers a test registry and counts the calls.
type countingGatherer struct {
	metrics *testMetrics
	calls   atomic.Int64
	// err, when set, is returned by every Gather instead of the registry.
	err error
}

// Gather implements [prometheus.Gatherer].
func (g *countingGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.calls.Add(1)
	if g.err != nil {
		return nil, g.err
	}

	return g.metrics.registry.Gather()
}

// mustDump returns the points of c for d, failing t on error.
func mustDump(t *testing.T, c *Collector, d port.Duration) []port.Point {
	t.Helper()

	return mustSnapshot(t, c, d).Points
}

// mustSnapshot returns the snapshot of c for d, failing t on error.
func mustSnapshot(t *testing.T, c *Collector, d port.Duration) port.Snapshot {
	t.Helper()

	snapshot, err := c.Dump(d)
	if err != nil {
		t.Fatal(err)
	}

	return snapshot
}

// waitFor polls cond until it holds, failing t after a second.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(time.Millisecond)
	}
}

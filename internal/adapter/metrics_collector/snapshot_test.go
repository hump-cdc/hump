package metricscollector

import (
	"slices"
	"testing"
	"time"

	"hump/internal/domain"
	port "hump/internal/port/metrics_collector"
)

func TestSnapshotReportsTopologyWhileIdle(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()}, WithTopology(testTopology()))
	t.Cleanup(c.close)

	snapshot := mustSnapshot(t, c, port.Duration1m)

	if len(snapshot.Points) != 0 {
		t.Fatalf("got %d points while idle", len(snapshot.Points))
	}

	if ops, ok := snapshot.Tables["public.users"]; !ok || counts(ops) != [4]float64{} {
		t.Fatalf("got tables %+v, want public.users with zero counts", snapshot.Tables)
	}

	if len(snapshot.Sinks) != 1 || len(snapshot.Sinks[0].Targets) != 2 {
		t.Fatalf("got sinks %+v, want one endpoint with two targets", snapshot.Sinks)
	}

	if len(snapshot.Connections) != 2 || snapshot.Connections[0].Success != 0 {
		t.Fatalf("got connections %+v, want both edges at zero", snapshot.Connections)
	}
}

func TestSnapshotCountsTableOperationsAbsolutely(t *testing.T) {
	m := newTestMetrics()
	c := newActiveCollector(t, m, testTopology())

	m.rows.WithLabelValues("insert", "public", "users").Observe(0.001)
	m.rows.WithLabelValues("insert", "public", "users").Observe(0.001)
	m.rows.WithLabelValues("update", "public", "users").Observe(0.001)
	m.rows.WithLabelValues("delete", "public", "users").Observe(0.001)
	m.rows.WithLabelValues("truncate", "public", "users").Observe(0.001)

	want := [4]float64{2, 1, 1, 1}
	waitFor(t, func() bool { return counts(mustSnapshot(t, c, port.Duration1m).Tables["public.users"]) == want })

	// The readings are absolute, so a further scrape must not double them.
	time.Sleep(30 * time.Millisecond)
	if got := counts(mustSnapshot(t, c, port.Duration1m).Tables["public.users"]); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got := counts(mustSnapshot(t, c, port.Duration1m).Tables["public.posts"]); got != [4]float64{} {
		t.Fatalf("got %v for an untouched table, want zero counts", got)
	}
}

func TestSnapshotReportsTableLatencyInMilliseconds(t *testing.T) {
	m := newTestMetrics()
	// The series has to exist before the first scrape, which is the baseline the
	// later observations are measured against.
	m.rows.WithLabelValues("insert", "public", "users").Observe(0.009)
	c := newActiveCollector(t, m, testTopology())
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })

	for range 20 {
		m.rows.WithLabelValues("insert", "public", "users").Observe(0.009)
	}

	waitFor(t, func() bool { return mustSnapshot(t, c, port.Duration1m).Tables["public.users"].Insert.P95 != nil })

	users := mustSnapshot(t, c, port.Duration1m).Tables["public.users"]
	if got := *users.Insert.P95; got != 9.75 {
		t.Fatalf("got insert p95 %v ms, want 9.75", got)
	}

	if users.Update.P95 != nil {
		t.Fatalf("got update p95 %v for an untouched operation, want none", *users.Update.P95)
	}
}

func TestSnapshotCountsSinkOperationsWithoutRetries(t *testing.T) {
	m := newTestMetrics()
	c := newActiveCollector(t, m, testTopology())

	m.messages.WithLabelValues("success", "kafka", "h", "users").Add(49)
	m.messages.WithLabelValues("failure", "kafka", "h", "users").Add(1)
	m.messages.WithLabelValues("retry", "kafka", "h", "users").Add(7)
	m.messages.WithLabelValues("success", "kafka", "h", "posts").Add(50)

	waitFor(t, func() bool { return mustSnapshot(t, c, port.Duration1m).Sinks[0].Operations == 100 })

	sink := mustSnapshot(t, c, port.Duration1m).Sinks[0]
	if sink.Success == nil || *sink.Success != 0.99 {
		t.Fatalf("got success %v, want 0.99", sink.Success)
	}

	users := sink.Targets[1]
	if users.Target != "users" || users.Operations != 50 || *users.Success != 0.98 {
		t.Fatalf("got target %+v, want users at 50 and 0.98", users)
	}
}

func TestSnapshotReportsLatencyInMilliseconds(t *testing.T) {
	m := newTestMetrics()
	// The series has to exist before the first scrape, which is the baseline the
	// later observations are measured against.
	m.duration.WithLabelValues("kafka", "h", "users").Observe(0.009)
	c := newActiveCollector(t, m, testTopology())
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })

	for range 20 {
		m.duration.WithLabelValues("kafka", "h", "users").Observe(0.009)
	}

	waitFor(t, func() bool { return mustSnapshot(t, c, port.Duration1m).Sinks[0].P95 != nil })

	snapshot := mustSnapshot(t, c, port.Duration1m)
	if got := *snapshot.Sinks[0].P95; got != 9.75 {
		t.Fatalf("got p95 %v ms, want 9.75", got)
	}

	// The untouched target has no observations of its own.
	if snapshot.Sinks[0].Targets[0].P95 != nil {
		t.Fatalf("got p95 %v for an untouched target, want none", *snapshot.Sinks[0].Targets[0].P95)
	}

	points := snapshot.Points
	if got := points[len(points)-1].SinkLatencyP95["kafka"]; got != 9.75 {
		t.Fatalf("got point p95 %v ms, want 9.75", got)
	}
}

func TestSnapshotFoldsPointLatencyBySinkType(t *testing.T) {
	m := newTestMetrics()
	m.duration.WithLabelValues("kafka", "h", "users").Observe(0.009)
	m.duration.WithLabelValues("kafka", "h", "posts").Observe(0.009)
	c := newActiveCollector(t, m, testTopology())
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })

	for range 20 {
		m.duration.WithLabelValues("kafka", "h", "users").Observe(0.009)
		m.duration.WithLabelValues("kafka", "h", "posts").Observe(0.009)
	}

	waitFor(t, func() bool { return mustSnapshot(t, c, port.Duration1m).Sinks[0].P95 != nil })

	snapshot := mustSnapshot(t, c, port.Duration1m)
	points := snapshot.Points
	if latency := points[len(points)-1].SinkLatencyP95; len(latency) != 1 || latency["kafka"] != 9.75 {
		t.Fatalf("got point latency %v, want one kafka entry", latency)
	}

	for _, target := range snapshot.Sinks[0].Targets {
		if target.P95 == nil || *target.P95 != 9.75 {
			t.Fatalf("got p95 %v for %q, want 9.75", target.P95, target.Target)
		}
	}
}

func TestRecordCountsConnectionsWithoutRetries(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()}, WithTopology(testTopology()))
	t.Cleanup(c.close)

	conn := testTopology()[0]
	c.Record(conn, domain.DeliveryStatusSuccess, 24)
	c.Record(conn, domain.DeliveryStatusFailure, 1)
	c.Record(conn, domain.DeliveryStatusRetry, 7)

	connections := mustSnapshot(t, c, port.Duration1m).Connections
	users := connections[1]
	if users.Table != identifier("users") || users.Success != 24 || users.Failure != 1 {
		t.Fatalf("got %+v, want users at 24 successes and 1 failure", users)
	}

	if connections[0].Success != 0 || connections[0].Failure != 0 {
		t.Fatalf("got %+v, want posts at 0", connections[0])
	}
}

func TestRecordSurvivesIdleTimeout(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()},
		WithInterval(5*time.Millisecond), WithIdleTimeout(10*time.Millisecond), WithTopology(testTopology()))
	t.Cleanup(c.close)

	c.Connect()
	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) > 0 })
	c.Record(testTopology()[0], domain.DeliveryStatusSuccess, 3)
	c.Disconnect()

	waitFor(t, func() bool { return len(mustDump(t, c, port.Duration1m)) == 0 })

	if got := mustSnapshot(t, c, port.Duration1m).Connections[1].Success; got != 3 {
		t.Fatalf("got %v after deactivation, want 3", got)
	}
}

func TestSnapshotCollapsesTablesSharingATarget(t *testing.T) {
	// An http sink renders the same target for every table, so two edges land on
	// one target rather than reporting the same series twice.
	topology := []port.Connection{
		{Table: identifier("users"), SinkType: "http", Host: "h", Target: "POST"},
		{Table: identifier("posts"), SinkType: "http", Host: "h", Target: "POST"},
	}

	m := newTestMetrics()
	c := newActiveCollector(t, m, topology)

	m.messages.WithLabelValues("success", "http", "h", "POST").Add(10)
	waitFor(t, func() bool { return mustSnapshot(t, c, port.Duration1m).Sinks[0].Operations == 10 })

	snapshot := mustSnapshot(t, c, port.Duration1m)
	if len(snapshot.Sinks) != 1 || len(snapshot.Sinks[0].Targets) != 1 {
		t.Fatalf("got sinks %+v, want one endpoint with one target", snapshot.Sinks)
	}

	if got := snapshot.Sinks[0].Targets[0].Operations; got != 10 {
		t.Fatalf("got %v, want the shared series counted once", got)
	}

	// The collapse is at the target level; both edges are still listed.
	if len(snapshot.Connections) != 2 {
		t.Fatalf("got connections %+v, want both edges", snapshot.Connections)
	}
}

func TestSnapshotSortsSinksByTypeThenHost(t *testing.T) {
	topology := []port.Connection{
		{Table: identifier("users"), SinkType: "nats-core", Host: "b", Target: "hump.public.users"},
		{Table: identifier("users"), SinkType: "kafka", Host: "b", Target: "users"},
		{Table: identifier("posts"), SinkType: "kafka", Host: "a", Target: "posts"},
	}

	c := New(&countingGatherer{metrics: newTestMetrics()}, WithTopology(topology))
	t.Cleanup(c.close)

	got := make([]sinkEndpoint, 0, 3)
	for _, sink := range mustSnapshot(t, c, port.Duration1m).Sinks {
		got = append(got, sinkEndpoint{sinkType: sink.Type, host: sink.Host})
	}

	want := []sinkEndpoint{
		{sinkType: "kafka", host: "a"},
		{sinkType: "kafka", host: "b"},
		{sinkType: "nats-core", host: "b"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRecordIgnoredAfterClose(t *testing.T) {
	c := New(&countingGatherer{metrics: newTestMetrics()}, WithTopology(testTopology()))
	c.close()

	c.Record(testTopology()[0], domain.DeliveryStatusSuccess, 5)

	if got := mustSnapshot(t, c, port.Duration1m).Connections[1].Success; got != 0 {
		t.Fatalf("got %v after close, want 0", got)
	}
}

// testTopology is the configured edge set the snapshot tests report over: one
// kafka endpoint carrying two tables to two targets.
func testTopology() []port.Connection {
	return []port.Connection{
		{Table: identifier("users"), SinkType: "kafka", Host: "h", Target: "users"},
		{Table: identifier("posts"), SinkType: "kafka", Host: "h", Target: "posts"},
	}
}

// identifier returns the public table called name.
func identifier(name string) domain.TableIdentifier {
	return domain.TableIdentifier{Schema: "public", Name: domain.TableName(name)}
}

// newActiveCollector returns a connected collector over m and topology, scraping
// fast enough for a test to wait on.
func newActiveCollector(t *testing.T, m *testMetrics, topology []port.Connection) *Collector {
	t.Helper()

	c := New(&countingGatherer{metrics: m},
		WithInterval(5*time.Millisecond), WithIdleTimeout(time.Hour), WithTopology(topology))
	t.Cleanup(c.close)
	c.Connect()

	return c
}

// counts returns the operation counts of ops in insert, update, delete, truncate order.
func counts(ops port.TableOps) [4]float64 {
	return [4]float64{ops.Insert.Count, ops.Update.Count, ops.Delete.Count, ops.Truncate.Count}
}

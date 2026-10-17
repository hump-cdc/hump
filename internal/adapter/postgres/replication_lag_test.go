package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestReplicationLagPollerSetsRetainedWal(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return singleValue("19922944") })

	p := newTestReplicationLagPoller(t, server)

	if err := p.poll(context.Background()); err != nil {
		t.Fatalf("poll() error = %v", err)
	}

	if got := testutil.ToFloat64(p.metrics.replicationLag); got != 19922944 {
		t.Errorf("replicationLag = %v, want 19922944", got)
	}
}

func TestReplicationLagPollerBindsSlotName(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return singleValue("0") })

	p := newTestReplicationLagPoller(t, server)

	if err := p.poll(context.Background()); err != nil {
		t.Fatalf("poll() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 1 || !strings.Contains(statements[0], "pg_replication_slots") {
		t.Fatalf("statements = %q, want one pg_replication_slots query", statements)
	}

	params := server.boundParams()
	if len(params) != 1 || len(params[0]) != 1 || string(params[0][0]) != "hump_slot" {
		t.Errorf("bound params = %q, want [[hump_slot]]", params)
	}
}

func TestReplicationLagPollerReportsZeroForMissingSlot(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return queryResult{columns: 1} })

	p := newTestReplicationLagPoller(t, server)
	p.metrics.setReplicationLag(100)

	if err := p.poll(context.Background()); err != nil {
		t.Fatalf("poll() error = %v", err)
	}

	if got := testutil.ToFloat64(p.metrics.replicationLag); got != 0 {
		t.Errorf("replicationLag = %v, want 0", got)
	}
}

func TestReplicationLagPollerKeepsLastValueOnNull(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return queryResult{columns: 1, rows: [][][]byte{{nil}}} })

	p := newTestReplicationLagPoller(t, server)
	p.metrics.setReplicationLag(100)

	if err := p.poll(context.Background()); err == nil || !strings.Contains(err.Error(), "no confirmed flush position") {
		t.Fatalf("poll() error = %v, want it to mention the missing confirmed flush position", err)
	}

	if got := testutil.ToFloat64(p.metrics.replicationLag); got != 100 {
		t.Errorf("replicationLag = %v, want 100", got)
	}
}

func TestReplicationLagPollerKeepsLastValueOnServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })

	p := newTestReplicationLagPoller(t, server)
	p.metrics.setReplicationLag(100)

	if err := p.poll(context.Background()); err == nil || !strings.Contains(err.Error(), "query replication lag") {
		t.Fatalf("poll() error = %v, want it to mention the failed query", err)
	}

	if got := testutil.ToFloat64(p.metrics.replicationLag); got != 100 {
		t.Errorf("replicationLag = %v, want 100", got)
	}
}

func TestReplicationLagPollerRejectsUnparseableValue(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return singleValue("lots") })

	p := newTestReplicationLagPoller(t, server)

	if err := p.poll(context.Background()); err == nil || !strings.Contains(err.Error(), "parse replication lag") {
		t.Fatalf("poll() error = %v, want it to mention the unparseable value", err)
	}
}

func TestReplicationLagPollerRunsUntilContextIsCancelled(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return singleValue("42") })

	p := newTestReplicationLagPoller(t, server)
	p.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for len(server.recorded()) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for a second poll")
		}

		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for run to return")
	}

	if got := testutil.ToFloat64(p.metrics.replicationLag); got != 42 {
		t.Errorf("replicationLag = %v, want 42", got)
	}
}

func TestReplicationLagPollerKeepsRunningAfterFailedPoll(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })

	p := newTestReplicationLagPoller(t, server)
	p.interval = 10 * time.Millisecond
	p.metrics.setReplicationLag(100)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for len(server.recorded()) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for a poll after the failed one")
		}

		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for run to return")
	}

	if got := testutil.ToFloat64(p.metrics.replicationLag); got != 100 {
		t.Errorf("replicationLag = %v, want 100", got)
	}
}

// newTestReplicationLagPoller returns a poller for hump_slot on a fresh
// connection to s, with metrics on a private registry.
func newTestReplicationLagPoller(t *testing.T, s *fakeServer) *replicationLagPoller {
	t.Helper()

	return newReplicationLagPoller(newTestConnection(t, s), "hump_slot", newTestMetrics())
}

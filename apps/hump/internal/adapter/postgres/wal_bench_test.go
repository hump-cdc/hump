package postgres

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"hump/internal/domain"
)

// BenchmarkStartReceivesMessages measures the per-message cost of the receive
// loop. allocs/op is the signal to watch: ns/op is dominated by the loopback
// socket the fake server answers on.
func BenchmarkStartReceivesMessages(b *testing.B) {
	server := usersServer(b)
	w, _ := newReplicationWal(b, server, newTestConnection(b, server, withReplication()), configuredTable("public", "users"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newBenchCounter(b.N)
	errs := runStart(ctx, w, counter.handle)

	// The relation has to reach the cache before the timer starts: an insert
	// naming an unregistered relation ends the stream.
	server.stream(xLogData(0x10, 0x1000, relationMessage(1, "public", "users", idColumn())))
	server.stream(keepalive(0x1000, true))
	server.waitForStandbyUpdates(b, 1)

	insert := xLogData(0x100, 0x1000, insertMessage(1, textColumn("7")))

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		server.stream(insert)
	}
	counter.await(b, errs)

	b.StopTimer()

	cancel()
	<-errs
}

// benchCounter is a process handler that closes done once it has been
// dispatched want changes.
type benchCounter struct {
	want int
	seen atomic.Int64
	done chan struct{}
}

// newBenchCounter returns a counter that signals after want dispatches.
func newBenchCounter(want int) *benchCounter {
	return &benchCounter{want: want, done: make(chan struct{})}
}

// handle counts one dispatch. Start calls it from a single goroutine; the
// count is atomic only so that a failing await can report it.
func (c *benchCounter) handle(
	context.Context,
	time.Time,
	domain.OperationType,
	domain.DbName,
	domain.TableSchema,
	domain.TableName,
	domain.Headers,
	domain.DbRecord,
	domain.DbRecord,
) error {
	if c.seen.Add(1) == int64(c.want) {
		close(c.done)
	}

	return nil
}

// await blocks until every queued change has been dispatched, failing the
// benchmark if the stream ends first.
func (c *benchCounter) await(b *testing.B, errs <-chan error) {
	b.Helper()

	select {
	case <-c.done:
	case err := <-errs:
		b.Fatalf("Start() returned early: error = %v, dispatched %d of %d", err, c.seen.Load(), c.want)
	case <-time.After(time.Minute):
		b.Fatalf("timed out after %d of %d changes", c.seen.Load(), c.want)
	}
}

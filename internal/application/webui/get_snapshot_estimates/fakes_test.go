package getsnapshotestimates

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"hump/internal/domain"
	"hump/internal/port/database"

	"github.com/danielgtaylor/huma/v2/sse"
)

// call is one recorded GetRowCount call.
type call struct {
	table domain.TableIdentifier
	where string
}

// fakeDatabase answers GetRowCount from scripted counts and errors keyed by
// table name, recording every call and how many ran at once.
type fakeDatabase struct {
	database.Database

	counts map[string][]database.RowCount
	errs   map[string]error
	delay  time.Duration // delay is how long each count runs before it is streamed.
	block  chan struct{} // block, when set, holds every count until it is closed or ctx is done.

	mu     sync.Mutex
	calls  []call
	active int
	peak   int
}

// newFakeDatabase returns a database with nothing scripted.
func newFakeDatabase() *fakeDatabase {
	return &fakeDatabase{counts: map[string][]database.RowCount{}, errs: map[string]error{}}
}

// GetRowCount records the call and streams the counts scripted for the table.
func (d *fakeDatabase) GetRowCount(ctx context.Context, table domain.TableIdentifier, where string) (<-chan database.RowCount, error) {
	d.mu.Lock()
	d.calls = append(d.calls, call{table: table, where: where})
	err := d.errs[string(table.Name)]
	counts := d.counts[string(table.Name)]
	d.mu.Unlock()

	if err != nil {
		return nil, err
	}

	d.track(1)

	// Buffered like the adapter's, so a sender never waits on a reader.
	ch := make(chan database.RowCount, len(counts))
	go func() {
		defer close(ch)
		defer d.track(-1)

		time.Sleep(d.delay)

		if d.block != nil {
			select {
			case <-d.block:
			case <-ctx.Done():
				return
			}
		}

		for _, count := range counts {
			ch <- count
		}
	}()

	return ch, nil
}

// track adjusts the running count by delta and keeps its peak.
func (d *fakeDatabase) track(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.active += delta
	d.peak = max(d.peak, d.active)
}

// recordedCalls returns a copy of the calls made so far.
func (d *fakeDatabase) recordedCalls() []call {
	d.mu.Lock()
	defer d.mu.Unlock()

	return slices.Clone(d.calls)
}

// peakActive returns the most counts that ran at once.
func (d *fakeDatabase) peakActive() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.peak
}

// fakeSender records every payload sent, failing every send past failAfter
// when that is set.
type fakeSender struct {
	mu        sync.Mutex
	sent      []any
	failAfter int
}

// send is the sse.Sender the handler writes to.
func (s *fakeSender) send(message sse.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent = append(s.sent, message.Data)
	if s.failAfter > 0 && len(s.sent) > s.failAfter {
		return errors.New("client gone")
	}

	return nil
}

// events returns a copy of the payloads sent so far.
func (s *fakeSender) events() []any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.sent)
}

// serve runs the handler over tables to completion, failing the test if it
// takes longer than five seconds.
func serve(t *testing.T, c *Controller, sender *fakeSender, tables ...EstimateTable) {
	t.Helper()

	input := &EstimatesRequest{}
	input.Body.Tables = tables

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Handler()(context.Background(), input, sender.send)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return")
	}
}

// collect reads estimates until it is closed, failing the test if that takes
// longer than five seconds.
func collect(t *testing.T, estimates <-chan Estimate) []Estimate {
	t.Helper()

	var got []Estimate
	timeout := time.After(5 * time.Second)

	for {
		select {
		case estimate, ok := <-estimates:
			if !ok {
				return got
			}
			got = append(got, estimate)
		case <-timeout:
			t.Fatal("timed out waiting for the estimates to close")
		}
	}
}

// waitFor polls condition until it holds, failing the test after five seconds.
func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the condition")
		}

		time.Sleep(time.Millisecond)
	}
}

// scriptTables scripts n tables named t0, t1, … to answer counts, and returns
// the requests for them.
func scriptTables(db *fakeDatabase, n int, counts ...database.RowCount) []Request {
	requests := make([]Request, 0, n)
	for i := range n {
		name := fmt.Sprintf("t%d", i)
		db.counts[name] = counts
		requests = append(requests, request("public", name, ""))
	}

	return requests
}

// identifier builds a table identifier.
func identifier(schema, name string) domain.TableIdentifier {
	return domain.TableIdentifier{Schema: domain.TableSchema(schema), Name: domain.TableName(name)}
}

// request builds one use case request.
func request(schema, name, where string) Request {
	return Request{Table: identifier(schema, name), Where: where}
}

// table builds one requested table as the client sends it.
func table(schema, name, where string) EstimateTable {
	return EstimateTable{Schema: schema, Table: name, Where: where}
}

// filter returns the estimates of table, in order.
func filter(estimates []Estimate, table domain.TableIdentifier) []Estimate {
	var result []Estimate
	for _, estimate := range estimates {
		if estimate.Table == table {
			result = append(result, estimate)
		}
	}

	return result
}

// sameElements reports whether got and want hold the same elements, in any order.
func sameElements[T comparable](got, want []T) bool {
	if len(got) != len(want) {
		return false
	}

	rest := slices.Clone(got)
	for _, w := range want {
		i := slices.Index(rest, w)
		if i < 0 {
			return false
		}
		rest = slices.Delete(rest, i, i+1)
	}

	return true
}

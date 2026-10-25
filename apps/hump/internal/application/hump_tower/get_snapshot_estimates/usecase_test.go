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
)

func TestEstimateStreamsEveryCountInOrderPerTable(t *testing.T) {
	db := newFakeDatabase()
	db.counts["users"] = []database.RowCount{{Num: 100, Rough: true}, {Num: 98}}
	db.counts["orders"] = []database.RowCount{{Num: 7, Rough: true}}

	got := collect(t, NewUseCase(db).Estimate(context.Background(), []Request{
		request("public", "users", "id > 1"),
		request("", "orders", ""),
	}))

	users := identifier("public", "users")
	want := []Estimate{
		{Table: users, Count: 100, Rough: true},
		{Table: users, Count: 98},
		{Table: identifier("", "orders"), Count: 7, Rough: true},
	}
	if !sameElements(got, want) {
		t.Fatalf("estimates = %+v, want %+v in any order across tables", got, want)
	}

	if ofUsers := filter(got, users); !slices.Equal(ofUsers, want[:2]) {
		t.Errorf("users estimates = %+v, want the rough one before the exact one", ofUsers)
	}

	wantCalls := []call{
		{table: users, where: "id > 1"},
		{table: identifier("", "orders"), where: ""},
	}
	if calls := db.recordedCalls(); !sameElements(calls, wantCalls) {
		t.Errorf("GetRowCount calls = %+v, want %+v", calls, wantCalls)
	}
}

func TestEstimateReportsFailures(t *testing.T) {
	explainErr := errors.New("explain failed")

	db := newFakeDatabase()
	db.errs["missing"] = database.ErrTableNotFound
	db.counts["broken"] = []database.RowCount{{Error: explainErr}}

	got := collect(t, NewUseCase(db).Estimate(context.Background(), []Request{
		request("public", "missing", ""),
		request("public", "broken", ""),
	}))

	want := []Estimate{
		{Table: identifier("public", "missing"), Error: database.ErrTableNotFound},
		{Table: identifier("public", "broken"), Error: explainErr},
	}
	if !sameElements(got, want) {
		t.Fatalf("estimates = %+v, want %+v", got, want)
	}
}

func TestEstimateClosesAtOnceWithoutWork(t *testing.T) {
	tests := []struct {
		name     string
		useCase  *UseCase
		requests []Request
	}{
		{name: "no database", useCase: NewUseCase(nil), requests: []Request{request("public", "users", "")}},
		{name: "no tables", useCase: NewUseCase(newFakeDatabase())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := collect(t, tt.useCase.Estimate(context.Background(), tt.requests)); len(got) != 0 {
				t.Fatalf("estimates = %+v, want none", got)
			}
		})
	}
}

func TestEstimateCapsConcurrentCounts(t *testing.T) {
	db := newFakeDatabase()
	db.delay = 20 * time.Millisecond

	requests := scriptTables(db, 3*defaultMaxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

	got := collect(t, NewUseCase(db).Estimate(context.Background(), requests))

	if peak := db.peakActive(); peak != defaultMaxConcurrentEstimates {
		t.Errorf("peak concurrent counts = %d, want %d", peak, defaultMaxConcurrentEstimates)
	}

	if len(got) != len(requests) {
		t.Errorf("got %d estimates, want one per table", len(got))
	}
}

func TestEstimateCapsConcurrentCountsAtTheOption(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "positive", limit: 2, want: 2},
		{name: "non-positive keeps the default", limit: 0, want: defaultMaxConcurrentEstimates},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newFakeDatabase()
			db.delay = 20 * time.Millisecond

			requests := scriptTables(db, 3*defaultMaxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

			got := collect(t, NewUseCase(db, WithMaxConcurrentEstimates(tc.limit)).Estimate(context.Background(), requests))

			if peak := db.peakActive(); peak != tc.want {
				t.Errorf("peak concurrent counts = %d, want %d", peak, tc.want)
			}

			if len(got) != len(requests) {
				t.Errorf("got %d estimates, want one per table", len(got))
			}
		})
	}
}

func TestEstimateStopsWhenCancelled(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	t.Cleanup(func() { close(db.block) })

	requests := scriptTables(db, 3*defaultMaxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

	ctx, cancel := context.WithCancel(context.Background())
	estimates := NewUseCase(db).Estimate(ctx, requests)

	waitFor(t, func() bool { return db.peakActive() == defaultMaxConcurrentEstimates })
	cancel()

	if got := collect(t, estimates); len(got) != 0 {
		t.Errorf("estimates = %+v, want none after cancellation", got)
	}

	if calls := len(db.recordedCalls()); calls != defaultMaxConcurrentEstimates {
		t.Errorf("GetRowCount called %d times, want only the %d already running", calls, defaultMaxConcurrentEstimates)
	}
}

func TestEstimateDoesNothingWhenAlreadyCancelled(t *testing.T) {
	db := newFakeDatabase()
	requests := scriptTables(db, 3*defaultMaxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := collect(t, NewUseCase(db).Estimate(ctx, requests)); len(got) != 0 {
		t.Errorf("estimates = %+v, want none", got)
	}

	if calls := db.recordedCalls(); len(calls) != 0 {
		t.Errorf("GetRowCount calls = %+v, want none for a cancelled request", calls)
	}
}

func TestEstimateStopsAReaderlessTable(t *testing.T) {
	db := newFakeDatabase()
	db.counts["users"] = []database.RowCount{{Num: 1, Rough: true}, {Num: 1}}

	ctx, cancel := context.WithCancel(context.Background())
	estimates := NewUseCase(db).Estimate(ctx, []Request{request("public", "users", "")})

	<-estimates
	cancel()

	// The second estimate is either dropped or already handed over; either
	// way the stream must close rather than wait for a reader.
	collect(t, estimates)
}

func TestEstimateStopsAReaderlessFailure(t *testing.T) {
	db := newFakeDatabase()
	db.errs["missing"] = database.ErrTableNotFound

	ctx, cancel := context.WithCancel(context.Background())
	estimates := NewUseCase(db).Estimate(ctx, []Request{request("public", "missing", "")})

	waitFor(t, func() bool { return len(db.recordedCalls()) == 1 })
	cancel()

	collect(t, estimates)
}

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

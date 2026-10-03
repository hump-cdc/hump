package getsnapshotestimates

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

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

	requests := scriptTables(db, 3*maxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

	got := collect(t, NewUseCase(db).Estimate(context.Background(), requests))

	if peak := db.peakActive(); peak != maxConcurrentEstimates {
		t.Errorf("peak concurrent counts = %d, want %d", peak, maxConcurrentEstimates)
	}

	if len(got) != len(requests) {
		t.Errorf("got %d estimates, want one per table", len(got))
	}
}

func TestEstimateStopsWhenCancelled(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	t.Cleanup(func() { close(db.block) })

	requests := scriptTables(db, 3*maxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

	ctx, cancel := context.WithCancel(context.Background())
	estimates := NewUseCase(db).Estimate(ctx, requests)

	waitFor(t, func() bool { return db.peakActive() == maxConcurrentEstimates })
	cancel()

	if got := collect(t, estimates); len(got) != 0 {
		t.Errorf("estimates = %+v, want none after cancellation", got)
	}

	if calls := len(db.recordedCalls()); calls != maxConcurrentEstimates {
		t.Errorf("GetRowCount called %d times, want only the %d already running", calls, maxConcurrentEstimates)
	}
}

func TestEstimateDoesNothingWhenAlreadyCancelled(t *testing.T) {
	db := newFakeDatabase()
	requests := scriptTables(db, 3*maxConcurrentEstimates, database.RowCount{Num: 1, Rough: true})

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

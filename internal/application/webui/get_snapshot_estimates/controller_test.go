package getsnapshotestimates

import (
	"context"
	"errors"
	"testing"
	"time"

	"hump/internal/port/database"
)

func TestHandlerStreamsEventsThenDone(t *testing.T) {
	db := newFakeDatabase()
	db.counts["users"] = []database.RowCount{{Num: 100, Rough: true}, {Num: 98}}
	db.errs["missing"] = database.ErrTableNotFound
	db.counts["broken"] = []database.RowCount{{Error: errors.New("explain failed")}}

	sender := &fakeSender{}
	serve(t, NewController(NewUseCase(db)), sender,
		table("public", "users", "id > 1"),
		table("", "missing", ""),
		table("public", "broken", ""),
	)

	events := sender.events()
	if last := events[len(events)-1]; last != (EstimatesDoneEvent{}) {
		t.Fatalf("last event = %#v, want done", last)
	}

	want := []any{
		EstimateEvent{Schema: "public", Table: "users", Count: 100, Rough: true},
		EstimateEvent{Schema: "public", Table: "users", Count: 98},
		EstimateErrorEvent{Table: "missing", Error: database.ErrTableNotFound.Error()},
		EstimateErrorEvent{Schema: "public", Table: "broken", Error: "explain failed"},
	}
	if got := events[:len(events)-1]; !sameElements(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}

	if calls := db.recordedCalls(); !sameElements(calls, []call{
		{table: identifier("public", "users"), where: "id > 1"},
		{table: identifier("", "missing"), where: ""},
		{table: identifier("public", "broken"), where: ""},
	}) {
		t.Errorf("GetRowCount calls = %+v, want every requested table with its filter", calls)
	}
}

func TestHandlerSendsOnlyDoneWithoutDatabase(t *testing.T) {
	sender := &fakeSender{}
	serve(t, NewController(NewUseCase(nil)), sender, table("public", "users", ""))

	if events := sender.events(); len(events) != 1 || events[0] != (EstimatesDoneEvent{}) {
		t.Fatalf("events = %#v, want only done", events)
	}
}

func TestHandlerStopsOnceTheClientIsGone(t *testing.T) {
	db := newFakeDatabase()
	db.delay = 10 * time.Millisecond

	tables := make([]EstimateTable, 0, 3*maxConcurrentEstimates)
	for _, r := range scriptTables(db, 3*maxConcurrentEstimates, database.RowCount{Num: 1, Rough: true}, database.RowCount{Num: 1}) {
		tables = append(tables, table(string(r.Table.Schema), string(r.Table.Name), r.Where))
	}

	sender := &fakeSender{failAfter: 1}
	serve(t, NewController(NewUseCase(db)), sender, tables...)

	if events := sender.events(); len(events) != 2 {
		t.Fatalf("send was attempted %d times, want the first event and the failing one only", len(events))
	}

	if calls := len(db.recordedCalls()); calls >= len(tables) {
		t.Errorf("GetRowCount called %d times, want the queued tables abandoned after the failed send", calls)
	}
}

func TestHandlerStopsWhenTheRequestIsCancelled(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	t.Cleanup(func() { close(db.block) })
	db.counts["users"] = []database.RowCount{{Num: 1, Rough: true}}

	ctx, cancel := context.WithCancel(context.Background())
	sender := &fakeSender{}

	done := make(chan struct{})
	go func() {
		defer close(done)

		input := &EstimatesRequest{}
		input.Body.Tables = []EstimateTable{table("public", "users", "")}
		NewController(NewUseCase(db)).Handler()(ctx, input, sender.send)
	}()

	waitFor(t, func() bool { return len(db.recordedCalls()) == 1 })
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after the request was cancelled")
	}

	if events := sender.events(); len(events) != 1 || events[0] != (EstimatesDoneEvent{}) {
		t.Errorf("events = %#v, want only done after cancellation", events)
	}
}

func TestEventsNamesEveryPayload(t *testing.T) {
	events := NewController(nil).Events()

	want := map[string]any{
		"estimate": EstimateEvent{},
		"error":    EstimateErrorEvent{},
		"done":     EstimatesDoneEvent{},
	}
	for name, payload := range want {
		if events[name] != payload {
			t.Errorf("Events()[%q] = %#v, want %#v", name, events[name], payload)
		}
	}

	if len(events) != len(want) {
		t.Errorf("Events() has %d entries, want %d", len(events), len(want))
	}
}

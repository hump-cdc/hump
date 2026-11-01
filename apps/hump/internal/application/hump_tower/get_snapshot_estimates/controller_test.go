package getsnapshotestimates

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"hump/internal/port/database"

	"github.com/danielgtaylor/huma/v2/sse"
)

func TestHandlerStreamsEventsThenDone(t *testing.T) {
	db := newFakeDatabase()
	db.counts["users"] = []database.RowCount{{Num: 100, Rough: true}, {Num: 98}}
	db.errs["missing"] = database.ErrTableNotFound
	db.counts["broken"] = []database.RowCount{{Error: errors.New("explain failed")}}

	sender := &fakeSender{}
	serve(t, NewController(NewUseCase(testConfiguration(), db)), sender,
		table("", "users", "id > 1"),
		table("", "missing", ""),
		table("", "broken", ""),
		table("pg_catalog", "pg_authid", ""),
	)

	events := sender.events()
	if last := events[len(events)-1]; last != (EstimatesDoneEvent{}) {
		t.Fatalf("last event = %#v, want done", last)
	}

	want := []any{
		EstimateEvent{Table: "users", Count: 100, Rough: true},
		EstimateEvent{Table: "users", Count: 98},
		EstimateErrorEvent{Table: "missing", Error: database.ErrTableNotFound.Error()},
		EstimateErrorEvent{Table: "broken", Error: "explain failed"},
		EstimateErrorEvent{Schema: "pg_catalog", Table: "pg_authid", Error: ErrTableNotConfigured.Error()},
	}
	if got := events[:len(events)-1]; !sameElements(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}

	if calls := db.recordedCalls(); !sameElements(calls, []call{
		{table: identifier("public", "users"), where: "id > 1"},
		{table: identifier("public", "missing"), where: ""},
		{table: identifier("public", "broken"), where: ""},
	}) {
		t.Errorf("GetRowCount calls = %+v, want every requested table with its filter", calls)
	}
}

func TestHandlerSendsOnlyDoneWithoutDatabase(t *testing.T) {
	sender := &fakeSender{}
	serve(t, NewController(NewUseCase(testConfiguration(), nil)), sender, table("public", "users", ""))

	if events := sender.events(); len(events) != 1 || events[0] != (EstimatesDoneEvent{}) {
		t.Fatalf("events = %#v, want only done", events)
	}
}

func TestHandlerStopsOnceTheClientIsGone(t *testing.T) {
	db := newFakeDatabase()
	db.delay = 10 * time.Millisecond

	tables := make([]EstimateTable, 0, 3*defaultMaxConcurrentEstimates)
	for _, r := range scriptTables(db, 3*defaultMaxConcurrentEstimates, database.RowCount{Num: 1, Rough: true}, database.RowCount{Num: 1}) {
		tables = append(tables, table(string(r.Table.Schema), string(r.Table.Name), r.Where))
	}

	sender := &fakeSender{failAfter: 1}
	serve(t, NewController(NewUseCase(testConfiguration(), db)), sender, tables...)

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
		NewController(NewUseCase(testConfiguration(), db)).Handler()(ctx, input, sender.send)
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

func TestHandlerReturnsBeforeCancelledCountCloses(t *testing.T) {
	db := &controlledDatabase{started: make(chan chan database.RowCount, 2)}
	uc := NewUseCase(testConfiguration(), db, WithMaxConcurrentEstimates(1))
	controller := NewController(uc)
	sender := &fakeSender{failAfter: 1}
	input := &EstimatesRequest{}
	input.Body.Tables = []EstimateTable{table("public", "users", "")}
	done := make(chan struct{})
	go func() {
		defer close(done)
		controller.Handler()(t.Context(), input, sender.send)
	}()
	counts := receive(t, db.started)
	for range 2 {
		select {
		case counts <- database.RowCount{Num: 1}:
		case <-time.After(5 * time.Second):
			t.Fatal("handler did not read count")
		}
	}
	receive(t, done)
	if slots := len(uc.slots); slots != 1 {
		t.Fatalf("occupied slots = %d, want 1 until counts closes", slots)
	}
	if events := sender.events(); len(events) != 2 {
		t.Fatalf("events = %#v, want only the successful and failed sends", events)
	}
	close(counts)
	next := uc.Estimate(t.Context(), []Request{request("public", "users", "")})
	close(receive(t, db.started))
	collect(t, next)
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

// table builds one requested table as the client sends it.
func table(schema, name, where string) EstimateTable {
	return EstimateTable{Schema: schema, Table: name, Where: where}
}

package streamsnapshot

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2/sse"
)

func TestHandlerSendsTheCurrentStateThenEveryChange(t *testing.T) {
	started := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	finished := started.Add(time.Minute)
	sinks := []snapshot.Sink{{Type: "nats-core", Host: "localhost:4222", Target: "hump.public.users", Count: 10}}

	runner := &fakeRunner{
		current: snapshot.State{Status: snapshot.StatusIdle},
		updates: make(chan snapshot.State, 2),
	}
	runner.updates <- snapshot.State{
		Status:    snapshot.StatusProcessing,
		StartedAt: started,
		Tables: []snapshot.Table{
			{Schema: "public", Name: "users", Status: snapshot.TableProcessing, Rows: 10, EstimatedRows: 100, Sinks: sinks},
		},
	}
	runner.updates <- snapshot.State{
		Status:     snapshot.StatusDone,
		StartedAt:  started,
		FinishedAt: finished,
		Tables: []snapshot.Table{
			{Schema: "public", Name: "users", Status: snapshot.TableError, Error: "stopped", Rows: 10, Sinks: sinks},
		},
	}
	close(runner.updates)

	sender := &fakeSender{}
	serve(t, NewController(NewUseCase(runner)), sender)

	runSinks := []SnapshotRunSink{{Type: "nats-core", Host: "localhost:4222", Target: "hump.public.users", Count: 10}}
	want := []any{
		SnapshotIdleEvent{State: "idle"},
		SnapshotProcessingEvent{State: "processing", StartedAt: started, Tables: []SnapshotProcessingTable{
			{Schema: "public", Name: "users", Status: "processing", Rows: 10, EstimatedRows: 100, Sinks: runSinks},
		}},
		SnapshotDoneEvent{State: "done", StartedAt: started, FinishedAt: finished, Tables: []SnapshotDoneTable{
			{Schema: "public", Name: "users", Status: "error", Error: "stopped", Rows: 10, Sinks: runSinks},
		}},
	}
	if got := sender.events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}

	if !runner.unsubscribed {
		t.Error("the handler returned without unsubscribing")
	}
}

func TestHandlerStopsWhenTheClientIsGone(t *testing.T) {
	runner := &fakeRunner{
		current: snapshot.State{Status: snapshot.StatusIdle},
		updates: make(chan snapshot.State),
	}

	sender := &fakeSender{fail: true}
	serve(t, NewController(NewUseCase(runner)), sender)

	if !runner.unsubscribed {
		t.Error("the handler returned without unsubscribing")
	}
}

func TestHandlerSendsIdleWithoutRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sender := &fakeSender{onSend: cancel}

	done := make(chan struct{})
	go func() {
		defer close(done)
		NewController(NewUseCase(nil)).Handler()(ctx, nil, sender.send)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return")
	}

	if got := sender.events(); !reflect.DeepEqual(got, []any{SnapshotIdleEvent{State: "idle"}}) {
		t.Fatalf("events = %#v, want only idle", got)
	}
}

// fakeRunner hands out current and updates under id 1 and records the
// unsubscribe; nothing else is reachable.
type fakeRunner struct {
	snapshot.Runner

	current      snapshot.State
	updates      chan snapshot.State
	unsubscribed bool
}

func (r *fakeRunner) Subscribe() (uint64, snapshot.State, <-chan snapshot.State) {
	return 1, r.current, r.updates
}

func (r *fakeRunner) Unsubscribe(id uint64) {
	if id == 1 {
		r.unsubscribed = true
	}
}

// fakeSender records every payload sent; fail makes every send fail and
// onSend runs after each one.
type fakeSender struct {
	mu     sync.Mutex
	sent   []any
	fail   bool
	onSend func()
}

// send is the sse.Sender the handler writes to.
func (s *fakeSender) send(message sse.Message) error {
	s.mu.Lock()
	s.sent = append(s.sent, message.Data)
	s.mu.Unlock()

	if s.onSend != nil {
		s.onSend()
	}

	if s.fail {
		return errors.New("client gone")
	}

	return nil
}

// events returns a copy of the payloads sent so far.
func (s *fakeSender) events() []any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]any(nil), s.sent...)
}

// serve runs the handler to completion, failing the test if it takes longer
// than five seconds.
func serve(t *testing.T, c *Controller, sender *fakeSender) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Handler()(context.Background(), nil, sender.send)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return")
	}
}

package streamupdates

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	livestream "hump/internal/port/live_stream"
	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2/sse"
)

func TestHandlerSendsWelcomeThenTheCurrentSnapshot(t *testing.T) {
	var log []string
	runner := &fakeRunner{
		log: &log,
		current: snapshot.State{Status: snapshot.StatusProcessing, Tables: []snapshot.Table{
			{CompletedRows: 42, EstimatedRows: 100},
		}},
		updates: make(chan snapshot.State),
	}

	ctx, cancel := context.WithCancel(context.Background())
	sender := &fakeSender{}
	sender.onSend = func() {
		if len(sender.events()) == 2 {
			cancel()
		}
	}

	serve(ctx, t, NewController(NewUseCase(nil, nil, nil, runner)), sender)

	want := []any{WelcomeEvent{}, SnapshotEvent{State: "processing", Progress: 42}}
	if got := sender.events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestHandlerSendsIdleWithoutRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sender := &fakeSender{}
	sender.onSend = func() {
		if len(sender.events()) == 2 {
			cancel()
		}
	}

	serve(ctx, t, NewController(NewUseCase(nil, nil, nil, nil)), sender)

	want := []any{WelcomeEvent{}, SnapshotEvent{State: "idle"}}
	if got := sender.events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestHandlerSendsASnapshotOnlyWhenStateOrPercentChanges(t *testing.T) {
	var log []string
	runner := &fakeRunner{
		log:     &log,
		current: snapshot.State{Status: snapshot.StatusIdle},
		updates: make(chan snapshot.State),
	}

	processing := func(completed, estimated uint64) snapshot.State {
		return snapshot.State{Status: snapshot.StatusProcessing, Tables: []snapshot.Table{
			{CompletedRows: completed, EstimatedRows: estimated},
		}}
	}

	// The channel is unbuffered, so each state is taken by the handler before
	// the next one is offered and the context is cancelled only after the last.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		runner.updates <- processing(0, 0)
		runner.updates <- processing(5, 1000)
		runner.updates <- processing(10, 1000)
		runner.updates <- processing(15, 1000)
		runner.updates <- snapshot.State{Status: snapshot.StatusDone}
		cancel()
	}()

	sender := &fakeSender{}
	serve(ctx, t, NewController(NewUseCase(nil, nil, nil, runner)), sender)

	want := []any{
		WelcomeEvent{},
		SnapshotEvent{State: "idle"},
		SnapshotEvent{State: "processing"},
		SnapshotEvent{State: "processing", Progress: 1},
		SnapshotEvent{State: "done"},
	}
	if got := sender.events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestHandlerKeepsStreamingAfterTheRunnerCloses(t *testing.T) {
	var log []string
	stream := &fakeStreamer{log: &log, batches: make(chan livestream.Batch)}
	runner := &fakeRunner{
		log:     &log,
		current: snapshot.State{Status: snapshot.StatusIdle},
		updates: make(chan snapshot.State),
	}
	close(runner.updates)

	go func() {
		stream.batches <- livestream.Batch{Operations: []livestream.Operation{{Schema: "public", Table: "users"}}}
		close(stream.batches)
	}()

	sender := &fakeSender{}
	serve(context.Background(), t, NewController(NewUseCase(nil, nil, stream, runner)), sender)

	want := []any{
		WelcomeEvent{},
		SnapshotEvent{State: "idle"},
		OperationsEvent{Operations: []OperationEvent{{Schema: "public", Table: "users"}}},
	}
	if got := sender.events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestHandlerStopsWhenTheClientIsGone(t *testing.T) {
	var log []string
	runner := &fakeRunner{
		log:     &log,
		current: snapshot.State{Status: snapshot.StatusIdle},
		updates: make(chan snapshot.State),
	}

	sender := &fakeSender{fail: true}
	serve(context.Background(), t, NewController(NewUseCase(nil, nil, nil, runner)), sender)

	if !slices.Contains(log, "snapshot unsubscribe 3") {
		t.Error("the handler returned without unsubscribing")
	}
}

func TestHandlerStopsWhenASendFails(t *testing.T) {
	processing := snapshot.State{Status: snapshot.StatusProcessing, Tables: []snapshot.Table{
		{CompletedRows: 1, EstimatedRows: 2},
	}}

	cases := []struct {
		name    string
		batches []livestream.Batch
		states  []snapshot.State
		failAt  int
	}{
		{name: "current snapshot", failAt: 2},
		{name: "operations", batches: []livestream.Batch{{}}, failAt: 3},
		{name: "snapshot change", states: []snapshot.State{processing}, failAt: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var log []string
			stream := &fakeStreamer{log: &log, batches: make(chan livestream.Batch, len(tc.batches))}
			for _, batch := range tc.batches {
				stream.batches <- batch
			}

			runner := &fakeRunner{
				log:     &log,
				current: snapshot.State{Status: snapshot.StatusIdle},
				updates: make(chan snapshot.State, len(tc.states)),
			}
			for _, state := range tc.states {
				runner.updates <- state
			}

			sender := &fakeSender{}
			sender.onSend = func() {
				if len(sender.events()) == tc.failAt {
					sender.fail = true
				}
			}

			serve(context.Background(), t, NewController(NewUseCase(nil, nil, stream, runner)), sender)

			if got := len(sender.events()); got != tc.failAt {
				t.Fatalf("sent %d events, want the handler to stop at the failed send %d", got, tc.failAt)
			}
		})
	}
}

func TestEventsNamesEveryPayload(t *testing.T) {
	events := NewController(nil).Events()

	want := map[string]any{
		"welcome":    WelcomeEvent{},
		"operations": OperationsEvent{},
		"snapshot":   SnapshotEvent{},
	}
	for name, payload := range want {
		if !reflect.DeepEqual(events[name], payload) {
			t.Errorf("Events()[%q] = %#v, want %#v", name, events[name], payload)
		}
	}

	if len(events) != len(want) {
		t.Errorf("Events() has %d entries, want %d", len(events), len(want))
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

// serve runs the handler under ctx to completion, failing the test if it
// takes longer than five seconds.
func serve(ctx context.Context, t *testing.T, c *Controller, sender *fakeSender) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Handler()(ctx, nil, sender.send)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return")
	}
}

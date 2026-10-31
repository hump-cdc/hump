package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestWalGatePassesCallsWhileOpen(t *testing.T) {
	gate := newWalGate()
	calls := 0

	handler := gate.wrap(context.Background(), countingHandler(&calls))
	if err := handler(context.Background(), time.Now(), domain.OperationTypeInsert, "db", "public", "users", nil, database.MessageMetadata{}, nil, nil); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestWalGatePreservesMessageMetadata(t *testing.T) {
	gate := newWalGate()
	recorder := newHandlerRecorder(nil)
	handler := gate.wrap(context.Background(), recorder.handle)
	metadata := database.MessageMetadata{MessageID: "message-id", SnapshotID: "snapshot-id"}
	if err := handler(context.Background(), time.Now(), domain.OperationTypeInsert, "db", "public", "users", nil, metadata, nil, nil); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	calls := recorder.recorded()
	if len(calls) != 1 || calls[0].metadata != metadata {
		t.Errorf("handler calls = %+v, want metadata %+v", calls, metadata)
	}
}

func TestWalGateHoldsCallsUntilResume(t *testing.T) {
	gate := newWalGate()
	if err := gate.pause(context.Background()); err != nil {
		t.Fatalf("pause() error = %v", err)
	}

	entered := make(chan error, 1)
	go func() { entered <- gate.enter(context.Background()) }()

	select {
	case <-entered:
		t.Fatal("enter() returned while the gate was paused")
	case <-time.After(20 * time.Millisecond):
	}

	gate.resume()

	select {
	case err := <-entered:
		if err != nil {
			t.Fatalf("enter() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("enter() did not return after resume")
	}
}

func TestWalGateResumeUndoesRepeatedPauses(t *testing.T) {
	gate := newWalGate()
	for range 2 {
		if err := gate.pause(context.Background()); err != nil {
			t.Fatalf("pause() error = %v", err)
		}
	}

	gate.resume()
	gate.resume()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := gate.enter(ctx); err != nil {
		t.Fatalf("enter() error = %v, want the gate open after one resume", err)
	}
}

func TestWalGatePauseWaitsForInFlightCall(t *testing.T) {
	gate := newWalGate()
	if err := gate.enter(context.Background()); err != nil {
		t.Fatalf("enter() error = %v", err)
	}

	paused := make(chan error, 1)
	go func() { paused <- gate.pause(context.Background()) }()

	select {
	case <-paused:
		t.Fatal("pause() returned while a call was in flight")
	case <-time.After(20 * time.Millisecond):
	}

	gate.leave()

	select {
	case err := <-paused:
		if err != nil {
			t.Fatalf("pause() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pause() did not return once the call left")
	}
}

func TestWalGatePauseEndsWithContext(t *testing.T) {
	gate := newWalGate()
	if err := gate.enter(context.Background()); err != nil {
		t.Fatalf("enter() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := gate.pause(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("pause() error = %v, want context.Canceled", err)
	}
}

func TestWalGateWrapEndsWaitOnStop(t *testing.T) {
	gate := newWalGate()
	if err := gate.pause(context.Background()); err != nil {
		t.Fatalf("pause() error = %v", err)
	}

	stop, cancel := context.WithCancel(context.Background())
	calls := 0
	handler := gate.wrap(stop, countingHandler(&calls))

	done := make(chan error, 1)
	go func() {
		done <- handler(context.WithoutCancel(stop), time.Now(), domain.OperationTypeInsert, "db", "public", "users", nil, database.MessageMetadata{}, nil, nil)
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handler() did not return once stop was cancelled")
	}

	if calls != 0 {
		t.Fatalf("calls = %d, want the change held back", calls)
	}
}

// countingHandler is a handler that counts its calls in calls.
func countingHandler(calls *int) func(context.Context, time.Time, domain.OperationType, domain.DbName, domain.TableSchema, domain.TableName, domain.Headers, database.MessageMetadata, domain.DbRecord, domain.DbRecord) error {
	return func(context.Context, time.Time, domain.OperationType, domain.DbName, domain.TableSchema, domain.TableName, domain.Headers, database.MessageMetadata, domain.DbRecord, domain.DbRecord) error {
		*calls++
		return nil
	}
}

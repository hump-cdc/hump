package cleanupdatabase

import (
	"context"
	"errors"
	"testing"

	"hump/internal/port/database"
)

func TestCleanupCleansTheDatabase(t *testing.T) {
	db := &fakeDatabase{}

	if err := NewUseCase(db).Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup() = %v, want nil", err)
	}

	if db.calls != 1 {
		t.Errorf("db.Cleanup calls = %d, want 1", db.calls)
	}
}

func TestCleanupReturnsTheDatabaseError(t *testing.T) {
	cleanupErr := errors.New("slot is active")
	db := &fakeDatabase{err: cleanupErr}

	if err := NewUseCase(db).Cleanup(context.Background()); !errors.Is(err, cleanupErr) {
		t.Fatalf("Cleanup() = %v, want %v", err, cleanupErr)
	}
}

// fakeDatabase counts Cleanup calls and fails them with err; every other port
// method panics through the embedded nil interface.
type fakeDatabase struct {
	database.Database

	err   error
	calls int
}

// Cleanup records the call and returns the scripted error.
func (d *fakeDatabase) Cleanup(_ context.Context) error {
	d.calls++
	return d.err
}

// GetPublicationName returns a fixed publication name for the log line.
func (d *fakeDatabase) GetPublicationName() string {
	return "hump_pub"
}

// GetSlotName returns a fixed slot name for the log line.
func (d *fakeDatabase) GetSlotName() string {
	return "hump_slot"
}

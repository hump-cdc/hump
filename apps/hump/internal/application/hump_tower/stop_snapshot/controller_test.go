package stopsnapshot

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2"
)

func TestHandlerStopsTheRun(t *testing.T) {
	runner := &fakeRunner{}

	if _, err := NewController(NewUseCase(runner)).Handler()(context.Background(), nil); err != nil {
		t.Fatalf("handler = %v", err)
	}

	if !runner.stopped {
		t.Fatal("the runner was not stopped")
	}
}

func TestHandlerMapsErrors(t *testing.T) {
	cases := []struct {
		name   string
		runner snapshot.Runner
		status int
	}{
		{"not running", &fakeRunner{err: snapshot.ErrNotRunning}, http.StatusConflict},
		{"no runner", nil, http.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewController(NewUseCase(tc.runner)).Handler()(context.Background(), nil)

			var statusErr huma.StatusError
			if !errors.As(err, &statusErr) || statusErr.GetStatus() != tc.status {
				t.Fatalf("handler = %v, want status %d", err, tc.status)
			}
		})
	}
}

func TestHandlerPassesOtherErrorsThrough(t *testing.T) {
	boom := errors.New("boom")

	_, err := NewController(NewUseCase(&fakeRunner{err: boom})).Handler()(context.Background(), nil)
	if !errors.Is(err, boom) {
		t.Fatalf("handler = %v, want %v", err, boom)
	}

	var statusErr huma.StatusError
	if errors.As(err, &statusErr) {
		t.Fatalf("handler = %v, want the error unmapped", err)
	}
}

// fakeRunner records Stop and answers err.
type fakeRunner struct {
	snapshot.Runner

	stopped bool
	err     error
}

func (r *fakeRunner) Stop() error {
	r.stopped = true

	return r.err
}

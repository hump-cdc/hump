package startsnapshot

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"hump/internal/domain"
	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2"
)

func TestHandlerStartsTheRequestedTables(t *testing.T) {
	runner := &fakeRunner{}

	if _, err := NewController(NewUseCase(testConfiguration(), runner)).Handler()(context.Background(), request(
		SnapshotStartTable{Schema: "public", Table: "users", Where: "id > 1"},
		SnapshotStartTable{Table: "orders"},
	)); err != nil {
		t.Fatalf("handler = %v", err)
	}

	want := []snapshot.Request{
		{Table: domain.TableIdentifier{Schema: "public", Name: "users"}, Where: "id > 1"},
		{Table: domain.TableIdentifier{Schema: "sales", Name: "orders"}},
	}
	if !reflect.DeepEqual(runner.started, want) {
		t.Fatalf("started = %+v, want %+v", runner.started, want)
	}
}

func TestHandlerMapsErrors(t *testing.T) {
	cases := []struct {
		name   string
		runner snapshot.Runner
		status int
	}{
		{"already running", &fakeRunner{err: snapshot.ErrAlreadyRunning}, http.StatusConflict},
		{"invalid where", &fakeRunner{err: snapshot.ErrInvalidWhere}, http.StatusUnprocessableEntity},
		{"no runner", nil, http.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewController(NewUseCase(testConfiguration(), tc.runner)).Handler()(context.Background(), request(SnapshotStartTable{Table: "users"}))

			var statusErr huma.StatusError
			if !errors.As(err, &statusErr) || statusErr.GetStatus() != tc.status {
				t.Fatalf("handler = %v, want status %d", err, tc.status)
			}
		})
	}
}

func TestHandlerPassesOtherErrorsThrough(t *testing.T) {
	boom := errors.New("boom")

	_, err := NewController(NewUseCase(testConfiguration(), &fakeRunner{err: boom})).Handler()(context.Background(), request(SnapshotStartTable{Table: "users"}))
	if !errors.Is(err, boom) {
		t.Fatalf("handler = %v, want %v", err, boom)
	}

	var statusErr huma.StatusError
	if errors.As(err, &statusErr) {
		t.Fatalf("handler = %v, want the error unmapped", err)
	}
}

func TestHandlerRejectsUnconfiguredTarget(t *testing.T) {
	runner := &fakeRunner{}
	_, err := NewController(NewUseCase(testConfiguration(), runner)).Handler()(context.Background(), request(
		SnapshotStartTable{Table: "users"},
		SnapshotStartTable{Schema: "private", Table: "users"},
	))
	var statusErr huma.StatusError
	if !errors.As(err, &statusErr) || statusErr.GetStatus() != http.StatusUnprocessableEntity || runner.calls != 0 {
		t.Fatalf("handler = %v, runner calls = %d; want 422 without runner calls", err, runner.calls)
	}
}

// fakeRunner records the tables it was started with and answers err.
type fakeRunner struct {
	snapshot.Runner

	started []snapshot.Request
	calls   int
	err     error
}

func (r *fakeRunner) Start(tables []snapshot.Request) error {
	r.calls++
	r.started = tables

	return r.err
}

// request wraps tables in a start request.
func request(tables ...SnapshotStartTable) *SnapshotStartRequest {
	input := &SnapshotStartRequest{}
	input.Body.Tables = tables

	return input
}

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

	if _, err := NewController(NewUseCase(runner)).Handler()(context.Background(), request(
		SnapshotStartTable{Schema: "public", Table: "users", Where: "id > 1"},
		SnapshotStartTable{Table: "orders"},
	)); err != nil {
		t.Fatalf("handler = %v", err)
	}

	want := []snapshot.Request{
		{Table: domain.TableIdentifier{Schema: "public", Name: "users"}, Where: "id > 1"},
		{Table: domain.TableIdentifier{Name: "orders"}},
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
		{"no runner", nil, http.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewController(NewUseCase(tc.runner)).Handler()(context.Background(), request(SnapshotStartTable{Table: "users"}))

			var statusErr huma.StatusError
			if !errors.As(err, &statusErr) || statusErr.GetStatus() != tc.status {
				t.Fatalf("handler = %v, want status %d", err, tc.status)
			}
		})
	}
}

// fakeRunner records the tables it was started with and answers err.
type fakeRunner struct {
	snapshot.Runner

	started []snapshot.Request
	err     error
}

func (r *fakeRunner) Start(tables []snapshot.Request) error {
	r.started = tables

	return r.err
}

// request wraps tables in a start request.
func request(tables ...SnapshotStartTable) *SnapshotStartRequest {
	input := &SnapshotStartRequest{}
	input.Body.Tables = tables

	return input
}

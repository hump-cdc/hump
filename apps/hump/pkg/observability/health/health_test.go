package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthStartsUnhealthy(t *testing.T) {
	h := newHealth(t)

	code, failures := probe(t, h)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", code, http.StatusServiceUnavailable)
	}
	for _, name := range []string{"Postgres", "WAL Sender"} {
		if failures[name] != ErrNotOperational.Error() {
			t.Errorf("failure of %q = %q, want %q", name, failures[name], ErrNotOperational)
		}
	}
}

func TestHealthHealthyWhenAllChecksMarked(t *testing.T) {
	h := newHealth(t)
	h.Postgres.MarkHealthy()
	h.WalSender.MarkHealthy()

	code, failures := probe(t, h)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if len(failures) != 0 {
		t.Errorf("failures = %v, want none", failures)
	}
}

func TestHealthMarkUnhealthy(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{
		"with cause":    {err: errors.New("connection refused"), want: "connection refused"},
		"without cause": {err: nil, want: ErrNotOperational.Error()},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := newHealth(t)
			h.Postgres.MarkHealthy()
			h.WalSender.MarkHealthy()
			h.WalSender.MarkUnhealthy(tt.err)

			code, failures := probe(t, h)
			if code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", code, http.StatusServiceUnavailable)
			}
			if failures["WAL Sender"] != tt.want {
				t.Errorf("failure = %q, want %q", failures["WAL Sender"], tt.want)
			}
			if _, ok := failures["Postgres"]; ok {
				t.Errorf("Postgres reported as failing: %v", failures)
			}
		})
	}
}

// newHealth builds the checks under test.
func newHealth(t *testing.T) *Health {
	t.Helper()

	h, err := New("hump-test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return h
}

// probe serves one request through the handler and returns the status code
// and the failures it reported.
func probe(t *testing.T, h *Health) (int, map[string]string) {
	t.Helper()

	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var body struct {
		Failures map[string]string `json:"failures"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	return rec.Code, body.Failures
}

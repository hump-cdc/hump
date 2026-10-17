package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCorsPassesThroughWithoutOrigins(t *testing.T) {
	tests := []struct {
		name   string
		cors   string
		method string
	}{
		{"unset", "", http.MethodGet},
		{"blank", "  ,  ", http.MethodGet},
		{"unset preflight", "", http.MethodOptions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CORS", tt.cors)

			reached := false
			handler := Cors()(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				reached = true
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/", nil)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			handler.ServeHTTP(rec, req)

			if !reached {
				t.Error("request did not reach the next handler")
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
			}
		})
	}
}

func TestCorsInstallsOnChiWhenUnset(t *testing.T) {
	// chi builds the middleware chain on the first route and panics on a nil
	// middleware, which used to take the process down at startup.
	t.Setenv("CORS", "")

	router := chi.NewMux()
	router.Use(Cors())
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
	}
}

func TestCorsAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CORS", "https://example.com, https://app.example.com")

	mw := Cors()
	if mw == nil {
		t.Fatal("Cors() = nil, want middleware when CORS is set")
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		origin string
		want   string
	}{
		{"first origin", "https://example.com", "https://example.com"},
		{"second origin", "https://app.example.com", "https://app.example.com"},
		{"unlisted origin", "https://evil.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCorsHandlesPreflight(t *testing.T) {
	t.Setenv("CORS", "https://example.com")

	handler := Cors()(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("preflight request must not reach the next handler")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
}

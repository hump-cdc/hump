package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsDisabledWhenUnset(t *testing.T) {
	t.Setenv("CORS", "")

	if Cors() != nil {
		t.Fatal("Cors() = non-nil, want nil when CORS is unset")
	}
}

func TestCorsDisabledWhenBlank(t *testing.T) {
	t.Setenv("CORS", "  ,  ")

	if Cors() != nil {
		t.Fatal("Cors() = non-nil, want nil when CORS has no usable origins")
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

package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testAssets(t *testing.T) fs.FS {
	t.Helper()
	assets, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		t.Fatalf("sub assets: %v", err)
	}
	return assets
}

func TestStaticHandlerStatus(t *testing.T) {
	handler := staticHandler(testAssets(t))

	tests := []struct {
		name string
		path string
		want int
	}{
		{"root", "/", http.StatusOK},
		{"known route", "/stats", http.StatusOK},
		{"known route trailing slash", "/stats/", http.StatusOK},
		{"existing asset", "/favicon.ico", http.StatusOK},
		{"unknown route", "/does-not-exist", http.StatusNotFound},
		{"missing asset", "/static/js/missing.js", http.StatusNotFound},
		{"routes manifest not served", "/routes.json", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("GET %s: status = %d, want %d", tt.path, rec.Code, tt.want)
			}
		})
	}
}

// A not-found response must still carry the SPA shell so the client renders the
// in-app NotFound page.
func TestStaticHandlerNotFoundServesShell(t *testing.T) {
	handler := staticHandler(testAssets(t))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), `id="app"`) {
		t.Errorf("404 body is not the SPA shell:\n%s", rec.Body.String())
	}
}

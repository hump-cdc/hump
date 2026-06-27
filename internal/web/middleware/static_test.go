package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html":         {Data: []byte(`<!doctype html><div id="app"></div>`)},
		"favicon.ico":        {Data: []byte("icon")},
		"routes.json":        {Data: []byte(`["/stats"]`)},
		"static/js/index.js": {Data: []byte("console.log(1)")},
	}
}

func TestStaticHandlerStatus(t *testing.T) {
	handler := (&static{assets: testAssets()}).handler()

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
	handler := (&static{assets: testAssets()}).handler()

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

// Hashed assets under static/ are served with a long-lived immutable cache.
func TestStaticHandlerCachesAssets(t *testing.T) {
	handler := (&static{assets: testAssets()}).handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/js/index.js", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}
}

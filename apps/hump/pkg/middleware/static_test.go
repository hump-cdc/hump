package middleware

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

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

func TestStaticHandlerSetsPolicyOnShell(t *testing.T) {
	handler := (&static{assets: testAssets()}).handler()

	sum := sha256.Sum256([]byte(testInlineScript))
	hash := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"

	for _, path := range []string{"/", "/does-not-exist"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			handler.ServeHTTP(rec, req)

			csp := rec.Header().Get("Content-Security-Policy")
			if !strings.Contains(csp, "script-src 'self' "+hash+";") {
				t.Errorf("GET %s: Content-Security-Policy = %q, want script-src with %s", path, csp, hash)
			}
			if strings.Count(csp, "'sha256-") != 1 {
				t.Errorf("GET %s: Content-Security-Policy = %q, want one hash: the external script must be skipped", path, csp)
			}
		})
	}
}

func TestStaticHandlerPolicyWithoutInlineScripts(t *testing.T) {
	assets := testAssets()
	assets["index.html"] = &fstest.MapFile{Data: []byte(`<!doctype html><script src="/static/js/index.js"></script><div id="app"></div>`)}
	handler := (&static{assets: assets}).handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "sha256-") || !strings.Contains(csp, "script-src 'self'; ") {
		t.Errorf("Content-Security-Policy = %q, want script-src 'self' with no hash", csp)
	}
}

func TestStaticHandlerNoPolicyOnAssets(t *testing.T) {
	handler := (&static{assets: testAssets()}).handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/js/index.js", nil)
	handler.ServeHTTP(rec, req)

	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("Content-Security-Policy = %q, want none on an asset", csp)
	}
}

func TestStaticInvalidPathNotFound(t *testing.T) {
	handler := Static(embed.FS{}, "../outside")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestStaticWithoutShellNotFound(t *testing.T) {
	handler := Static(embed.FS{}, ".")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if ct := rec.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want no SPA shell", ct)
	}
}

func TestStaticRoutesMalformedManifest(t *testing.T) {
	assets := testAssets()
	assets["routes.json"] = &fstest.MapFile{Data: []byte(`{"not":"a list"}`)}

	if routes := (&static{assets: assets}).routes(); len(routes) != 0 {
		t.Errorf("routes = %v, want none", routes)
	}
}

// testInlineScript is the inline script of the test shell, hashed by the policy.
const testInlineScript = "\n  document.documentElement.classList.add(\"dark\")\n"

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html": {Data: []byte(`<!doctype html><head><script>` + testInlineScript +
			`</script><script defer src="/static/js/index.js"></script></head><div id="app"></div>`)},
		"favicon.ico":        {Data: []byte("icon")},
		"routes.json":        {Data: []byte(`["/stats"]`)},
		"static/js/index.js": {Data: []byte("console.log(1)")},
	}
}

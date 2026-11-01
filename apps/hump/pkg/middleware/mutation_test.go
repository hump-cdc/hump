package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSnapshotMutationsContentType(t *testing.T) {
	t.Setenv("CORS", "")
	for _, tt := range []struct {
		name   string
		values []string
		want   int
	}{
		{"missing", nil, http.StatusUnsupportedMediaType},
		{"empty", []string{""}, http.StatusUnsupportedMediaType},
		{"plain", []string{"text/plain"}, http.StatusUnsupportedMediaType},
		{"form", []string{"application/x-www-form-urlencoded"}, http.StatusUnsupportedMediaType},
		{"multipart", []string{"multipart/form-data; boundary=test"}, http.StatusUnsupportedMediaType},
		{"suffix", []string{"application/problem+json"}, http.StatusUnsupportedMediaType},
		{"malformed", []string{"application/json; charset"}, http.StatusUnsupportedMediaType},
		{"multiple", []string{"application/json", "text/plain"}, http.StatusUnsupportedMediaType},
		{"json", []string{"application/json"}, http.StatusNoContent},
		{"parameters", []string{`Application/JSON; charset="utf-8"`}, http.StatusNoContent},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/snapshot/stop", nil)
			req.Header["Content-Type"] = tt.values
			req.Header.Set("Origin", "http://evil.test")
			if tt.want == http.StatusNoContent {
				req.Header.Del("Origin")
			}
			assertMutationStatus(t, SnapshotMutations(), req, tt.want)
		})
	}
}

func TestSnapshotMutationsOrigin(t *testing.T) {
	for _, tt := range []struct {
		name    string
		target  string
		origins []string
		cors    string
		want    int
	}{
		{"CLI", "http://localhost:3200", nil, "", http.StatusNoContent},
		{"empty", "http://localhost:3200", []string{""}, "", http.StatusForbidden},
		{"null", "http://localhost:3200", []string{"null"}, "null,*", http.StatusForbidden},
		{"same localhost port", "http://localhost:54321", []string{"http://localhost:54321"}, "", http.StatusNoContent},
		{"other localhost port", "http://localhost:54321", []string{"http://localhost:3000"}, "", http.StatusForbidden},
		{"TLS", "https://example.test", []string{"https://example.test"}, "", http.StatusNoContent},
		{"wrong scheme", "https://example.test", []string{"http://example.test"}, "", http.StatusForbidden},
		{"forwarded headers ignored", "http://localhost:3200", []string{"https://ui.test"}, "", http.StatusForbidden},
		{"IPv6", "http://[::1]:54321", []string{"http://[::1]:54321"}, "", http.StatusNoContent},
		{"explicit", "http://localhost:3200", []string{"https://ui.test:3000"}, " , https://other.test, https://ui.test:3000 ,", http.StatusNoContent},
		{"wildcard", "http://localhost:3200", []string{"https://evil.test"}, "*", http.StatusForbidden},
		{"wildcard pattern", "http://localhost:3200", []string{"https://ui.test"}, "https://*.test", http.StatusForbidden},
		{"wildcard origin", "http://localhost:3200", []string{"https://*.test"}, "https://*.test", http.StatusForbidden},
		{"multiple lines", "http://localhost:3200", []string{"http://localhost:3200", "http://localhost:3200"}, "", http.StatusForbidden},
		{"comma list", "http://localhost:3200", []string{"http://localhost:3200,http://evil.test"}, "", http.StatusForbidden},
		{"space list", "http://localhost:3200", []string{"http://localhost:3200 http://evil.test"}, "", http.StatusForbidden},
		{"path", "http://localhost:3200", []string{"https://ui.test/"}, "https://ui.test/", http.StatusForbidden},
		{"credentials", "http://localhost:3200", []string{"https://user@ui.test"}, "https://user@ui.test", http.StatusForbidden},
		{"query", "http://localhost:3200", []string{"https://ui.test?"}, "https://ui.test?", http.StatusForbidden},
		{"fragment", "http://localhost:3200", []string{"https://ui.test#"}, "https://ui.test#", http.StatusForbidden},
		{"opaque", "http://localhost:3200", []string{"https:ui.test"}, "https:ui.test", http.StatusForbidden},
		{"non HTTP", "http://localhost:3200", []string{"ftp://ui.test"}, "ftp://ui.test", http.StatusForbidden},
		{"missing host", "http://localhost:3200", []string{"https://"}, "https://", http.StatusForbidden},
		{"invalid escape", "http://localhost:3200", []string{"https://%xx"}, "https://%xx", http.StatusForbidden},
		{"invalid IPv6", "http://localhost:3200", []string{"https://[invalid]"}, "https://[invalid]", http.StatusForbidden},
		{"unbracketed IPv6", "http://localhost:3200", []string{"http://::1"}, "http://::1", http.StatusForbidden},
		{"empty port", "http://localhost:3200", []string{"https://ui.test:"}, "https://ui.test:", http.StatusForbidden},
		{"invalid port", "http://localhost:3200", []string{"https://ui.test:65536"}, "https://ui.test:65536", http.StatusForbidden},
		{"whitespace", "http://localhost:3200", []string{" https://ui.test"}, "https://ui.test", http.StatusForbidden},
		{"exact host case", "http://localhost:3200", []string{"http://LOCALHOST:3200"}, "", http.StatusForbidden},
		{"no default port normalization", "http://localhost", []string{"http://localhost:80"}, "", http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CORS", tt.cors)
			req := httptest.NewRequest(http.MethodPost, tt.target+"/api/snapshot/start", nil)
			req.Header.Set("Content-Type", "application/json")
			if tt.origins != nil {
				req.Header["Origin"] = tt.origins
			}
			req.Header.Set("Forwarded", "proto=https;host=ui.test")
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Forwarded-Host", "ui.test")
			assertMutationStatus(t, SnapshotMutations(), req, tt.want)
		})
	}
}

func TestSnapshotMutationsLeavesOtherRoutesUnchanged(t *testing.T) {
	t.Setenv("CORS", "")
	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/api/snapshot/start"},
		{http.MethodOptions, "/api/snapshot/stop"},
		{http.MethodPost, "/api/snapshot/estimate"},
		{http.MethodPost, "/api/snapshot/tables"},
		{http.MethodPost, "/api/metrics"},
		{http.MethodPost, "/api/operations"},
		{http.MethodGet, "/api/snapshot"},
	} {
		t.Run(tt.method+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Origin", "null")
			assertMutationStatus(t, SnapshotMutations(), req, http.StatusNoContent)
		})
	}
}

func TestSnapshotMutationsReadsCORSOnce(t *testing.T) {
	t.Setenv("CORS", "https://ui.test")
	guard := SnapshotMutations()
	t.Setenv("CORS", "https://evil.test")
	req := httptest.NewRequest(http.MethodPost, "/api/snapshot/stop", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://ui.test")
	assertMutationStatus(t, guard, req, http.StatusNoContent)
}

// assertMutationStatus checks the response and whether the guarded handler ran.
func assertMutationStatus(t *testing.T, guard func(http.Handler) http.Handler, req *http.Request, want int) {
	t.Helper()
	called := false
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != want || called != (want == http.StatusNoContent) {
		t.Fatalf("status = %d, called = %t; want %d, %t", resp.Code, called, want, want == http.StatusNoContent)
	}
}

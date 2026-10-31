package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/template"
	"hump/pkg/version"
)

func TestProcessSendsRenderedRequest(t *testing.T) {
	var (
		gotMethod, gotBody, gotPath string
		gotHeader                   nethttp.Header
	)
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotBody, gotPath, gotHeader = r.Method, string(body), r.URL.Path, r.Header
		w.WriteHeader(nethttp.StatusAccepted)
		_, _ = w.Write([]byte("ignored"))
	}))
	t.Cleanup(srv.Close)

	cfg := sinkcfg.HTTPSink{
		URL:    mustTemplate(t, srv.URL+"/{{ .Table }}"),
		Method: mustTemplate(t, "PUT"),
		Body:   mustTemplate(t, "{{ .Schema }}.{{ .Table }}"),
		Headers: map[string]*template.Template{
			"X-Table": mustTemplate(t, "{{ .Table }}"),
			"X-Empty": mustTemplate(t, ""),
		},
	}
	capture := &sink.Capture{}

	if err := Process(context.Background(), cfg, testContext(), capture); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if gotMethod != "PUT" || gotPath != "/users" || gotBody != "public.users" {
		t.Errorf("request = %s %s %q, want PUT /users %q", gotMethod, gotPath, gotBody, "public.users")
	}
	if got := gotHeader.Get("X-Table"); got != "users" {
		t.Errorf("X-Table = %q, want users", got)
	}
	if _, ok := gotHeader["X-Empty"]; ok {
		t.Error("X-Empty sent, want an empty rendered header dropped")
	}
	if got, want := gotHeader.Get("User-Agent"), "hump/"+version.Version; got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}

	if capture.Target != srv.URL+"/users" {
		t.Errorf("capture.Target = %q, want %q", capture.Target, srv.URL+"/users")
	}
	if string(capture.Payload) != "public.users" {
		t.Errorf("capture.Payload = %q, want public.users", capture.Payload)
	}
	wantMeta := map[string]string{"method": "PUT", "status": "202", "header.X-Table": "users"}
	for k, v := range wantMeta {
		if capture.Metadata[k] != v {
			t.Errorf("capture.Metadata[%q] = %q, want %q", k, capture.Metadata[k], v)
		}
	}
}

func TestProcessWithoutCapture(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := Process(context.Background(), sinkcfg.HTTPSink{URL: mustTemplate(t, srv.URL)}, testContext(), nil); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
}

func TestProcessClassifiesStatus(t *testing.T) {
	tests := []struct {
		code      int
		permanent bool
	}{
		{nethttp.StatusBadRequest, true},
		{nethttp.StatusNotFound, true},
		{nethttp.StatusRequestTimeout, false},
		{nethttp.StatusTooManyRequests, false},
		{nethttp.StatusInternalServerError, false},
		{nethttp.StatusServiceUnavailable, false},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.code), func(t *testing.T) {
			srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
				w.WriteHeader(tt.code)
			}))
			t.Cleanup(srv.Close)

			capture := &sink.Capture{}
			err := Process(context.Background(), sinkcfg.HTTPSink{URL: mustTemplate(t, srv.URL)}, testContext(), capture)

			var statusErr *statusError
			if !errors.As(err, &statusErr) || statusErr.code != tt.code {
				t.Fatalf("Process() error = %v, want statusError %d", err, tt.code)
			}
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
				t.Errorf("permanent = %v, want %v (err %v)", got, tt.permanent, err)
			}
			if got, want := capture.Metadata["status"], strconv.Itoa(tt.code); got != want {
				t.Errorf("capture status = %q, want %q", got, want)
			}
		})
	}
}

func TestProcessDoesNotFollowRedirects(t *testing.T) {
	for _, code := range []int{
		nethttp.StatusMovedPermanently,
		nethttp.StatusFound,
		nethttp.StatusSeeOther,
		nethttp.StatusTemporaryRedirect,
		nethttp.StatusPermanentRedirect,
	} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			var targetHits atomic.Int64
			target := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
				targetHits.Add(1)
				w.WriteHeader(nethttp.StatusOK)
			}))
			t.Cleanup(target.Close)
			srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				nethttp.Redirect(w, r, target.URL, code)
			}))
			t.Cleanup(srv.Close)

			cfg := sinkcfg.HTTPSink{
				URL:    mustTemplate(t, srv.URL),
				Method: mustTemplate(t, "POST"),
				Body:   mustTemplate(t, "public.users"),
			}
			capture := &sink.Capture{}
			err := Process(context.Background(), cfg, testContext(), capture)

			var statusErr *statusError
			if !errors.As(err, &statusErr) || statusErr.code != code {
				t.Errorf("Process() error = %v, want statusError %d", err, code)
			}
			if got := targetHits.Load(); got != 0 {
				t.Errorf("redirect target requests = %d, want 0", got)
			}
			if got, want := capture.Metadata["status"], strconv.Itoa(code); got != want {
				t.Errorf("capture status = %q, want %q", got, want)
			}
			if capture.Target != srv.URL {
				t.Errorf("capture.Target = %q, want %q", capture.Target, srv.URL)
			}
		})
	}
}

func TestProcessReturnsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Process(ctx, sinkcfg.HTTPSink{URL: mustTemplate(t, "http://127.0.0.1")}, testContext(), nil)
	if !errors.Is(err, context.Canceled) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want raw context.Canceled", err)
	}
}

func TestProcessRenderFailuresArePermanent(t *testing.T) {
	bad := "{{ .NoSuchField }}"
	tests := []struct {
		name string
		cfg  sinkcfg.HTTPSink
	}{
		{"url", sinkcfg.HTTPSink{URL: mustTemplate(t, bad)}},
		{"method", sinkcfg.HTTPSink{URL: mustTemplate(t, "http://x"), Method: mustTemplate(t, bad)}},
		{"body", sinkcfg.HTTPSink{URL: mustTemplate(t, "http://x"), Body: mustTemplate(t, bad)}},
		{"header", sinkcfg.HTTPSink{
			URL:     mustTemplate(t, "http://x"),
			Headers: map[string]*template.Template{"X-Bad": mustTemplate(t, bad)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Process(context.Background(), tt.cfg, testContext(), nil)
			if !errors.Is(err, template.ErrRender) || !errors.Is(err, sink.ErrSinkPermanent) {
				t.Fatalf("Process() error = %v, want permanent render error", err)
			}
		})
	}
}

func TestProcessUnbuildableRequestIsPermanent(t *testing.T) {
	cfg := sinkcfg.HTTPSink{URL: mustTemplate(t, "http://x"), Method: mustTemplate(t, "BAD METHOD")}

	err := Process(context.Background(), cfg, testContext(), nil)
	if !errors.Is(err, errBuildRequest) || !errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want permanent errBuildRequest", err)
	}
}

func TestProcessTransportFailureIsTransient(t *testing.T) {
	url := "http://" + closedAddr(t)
	capture := &sink.Capture{}

	err := Process(context.Background(), sinkcfg.HTTPSink{URL: mustTemplate(t, url)}, testContext(), capture)
	if err == nil || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want transient transport error", err)
	}
	if capture.Target != url {
		t.Errorf("capture.Target = %q, want %q", capture.Target, url)
	}
	if _, ok := capture.Metadata["status"]; ok {
		t.Error("capture has status, want none without a response")
	}
}

func TestProcessErrorsHideSecretURLs(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
		t.Run(level.String(), func(t *testing.T) {
			secret := "sentinel-secret"
			raw := "http://user:" + secret + "@example.com/" + secret + "?token=" + secret + "#" + secret
			dns := &net.DNSError{Err: secret, Name: secret, IsNotFound: true}
			original := defaultClient
			t.Cleanup(func() { defaultClient = original })
			tests := []struct {
				name, target string
				cause        error
				status       int
				permanent    bool
			}{
				{name: "build", target: raw + "%zz", permanent: true},
				{name: "transport", target: raw, cause: fmt.Errorf("%s: %w", secret, dns), permanent: true},
				{name: "context", target: raw, cause: fmt.Errorf("%s: %w", secret, context.Canceled)},
				{name: "status", target: raw, status: 401, permanent: true},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					defaultClient = &nethttp.Client{Transport: diagnosticTransport(func(req *nethttp.Request) (*nethttp.Response, error) {
						if tt.cause != nil {
							return nil, tt.cause
						}
						return &nethttp.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader("")), Header: make(nethttp.Header), Request: req}, nil
					})}
					capture := &sink.Capture{}
					err := Process(context.Background(), sinkcfg.HTTPSink{URL: mustTemplate(t, tt.target)}, testContext(), capture)
					if err == nil {
						t.Fatal("Process() succeeded")
					}
					assertSafeHTTPLog(t, level, err, secret)
					if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
						t.Fatalf("permanent = %v, want %v", got, tt.permanent)
					}
					if tt.cause != nil {
						if !errors.Is(err, tt.cause) {
							t.Fatal("original transport cause lost")
						}
						var urlErr *url.Error
						if !errors.As(err, &urlErr) || urlErr.URL == "" {
							t.Fatal("original URL error lost")
						}
						if capture.Target != tt.target {
							t.Fatal("capture target was changed")
						}
					}
					if tt.name == "transport" {
						var got *net.DNSError
						if !errors.As(err, &got) || got != dns {
							t.Fatal("DNS cause lost")
						}
					}
					if tt.name == "context" && !errors.Is(err, context.Canceled) {
						t.Fatal("context cause lost")
					}
					if tt.name == "build" && !errors.Is(err, errBuildRequest) {
						t.Fatal("build classification lost")
					}
				})
			}
		})
	}
}

func TestProcessRedirectErrorsHideBothURLsAndInnerCause(t *testing.T) {
	secret := "sentinel-secret"
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		nethttp.Redirect(w, r, "http://redirect.example/"+secret+"?token="+secret, nethttp.StatusFound)
	}))
	t.Cleanup(srv.Close)
	cause := errors.New("redirect denied: " + secret)
	original := defaultClient
	defaultClient = &nethttp.Client{CheckRedirect: func(_ *nethttp.Request, _ []*nethttp.Request) error { return cause }}
	t.Cleanup(func() { defaultClient = original })
	err := Process(context.Background(), sinkcfg.HTTPSink{URL: mustTemplate(t, srv.URL+"/"+secret)}, testContext(), nil)
	if !errors.Is(err, cause) {
		t.Fatal("redirect cause lost")
	}
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
		assertSafeHTTPLog(t, level, err, secret)
	}
}

func TestProcessBuildAndRenderErrorsHideSecretText(t *testing.T) {
	secret := "sentinel-secret"
	tests := []struct {
		name string
		cfg  sinkcfg.HTTPSink
	}{
		{"method", sinkcfg.HTTPSink{URL: mustTemplate(t, "https://example.com"), Method: mustTemplate(t, secret+" invalid")}},
		{"url template", sinkcfg.HTTPSink{URL: mustTemplate(t, "https://example.com/"+secret+"/{{ .Missing }}")}},
		{"header template", sinkcfg.HTTPSink{URL: mustTemplate(t, "https://example.com"), Headers: map[string]*template.Template{secret: mustTemplate(t, "{{ .Missing }}")}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Process(context.Background(), tt.cfg, testContext(), nil)
			if !errors.Is(err, sink.ErrSinkPermanent) {
				t.Fatal("permanent classification lost")
			}
			assertSafeHTTPLog(t, slog.LevelDebug, err, secret)
		})
	}
}

func TestIsPermanentHttpError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"dns not found", fmt.Errorf("do: %w", &net.DNSError{Name: "x", IsNotFound: true}), true},
		{"dns temporary", &net.DNSError{Name: "x", IsTemporary: true}, false},
		{"other", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPermanentHttpError(tt.err); got != tt.want {
				t.Errorf("isPermanentHttpError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsPermanentStatus(t *testing.T) {
	tests := map[int]bool{
		nethttp.StatusOK:                  false,
		nethttp.StatusFound:               false,
		nethttp.StatusUnauthorized:        true,
		nethttp.StatusRequestTimeout:      false,
		nethttp.StatusTooManyRequests:     false,
		nethttp.StatusInternalServerError: false,
	}
	for code, want := range tests {
		if got := isPermanentStatus(code); got != want {
			t.Errorf("isPermanentStatus(%d) = %v, want %v", code, got, want)
		}
	}
}

func TestStatusErrorMessage(t *testing.T) {
	err := &statusError{code: 503, url: "http://x"}

	if got, want := err.Error(), "http 503 from http://x"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// diagnosticTransport substitutes a transport failure without a network connection.
type diagnosticTransport func(*nethttp.Request) (*nethttp.Response, error)

// RoundTrip invokes the configured test transport.
func (f diagnosticTransport) RoundTrip(req *nethttp.Request) (*nethttp.Response, error) {
	return f(req)
}

// assertSafeHTTPLog checks the actual JSON serialization at the selected logging level.
func assertSafeHTTPLog(t *testing.T, level slog.Level, err error, secret string) {
	t.Helper()
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: level}))
	logger.Error("delivery failed", "err", err)
	logger.Debug("delivery failed", "err", err)
	if output.Len() == 0 || strings.Contains(output.String(), secret) || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe diagnostic: %s", output.String())
	}
}

// testContext returns a row change on public.users.
func testContext() *sink.Context {
	return &sink.Context{Schema: "public", Table: "users"}
}

// mustTemplate parses source or fails the test.
func mustTemplate(t *testing.T, source string) *template.Template {
	t.Helper()

	tmpl, err := template.New(source)
	if err != nil {
		t.Fatalf("template.New(%q) error = %v", source, err)
	}

	return tmpl
}

// closedAddr returns the address of a listener that is already closed, so a
// dial to it is refused immediately.
func closedAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return addr
}

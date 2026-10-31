package nats

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/template"
)

func TestProcessCorePublishesAndFlushes(t *testing.T) {
	srv := newFakeServer(t, nil)
	cfg := sinkcfg.NatsCoreSink{NatsCommon: sinkcfg.NatsCommon{
		URL:     mustTemplate(t, srv.url()),
		Name:    mustTemplate(t, "hump-test"),
		Subject: mustTemplate(t, "cdc.{{ .Table }}"),
		Payload: mustTemplate(t, "payload"),
		Headers: map[string]*template.Template{
			"X-Table": mustTemplate(t, "{{ .Table }}"),
			"X-Empty": mustTemplate(t, ""),
		},
	}}
	forgetClients(t, srv.url())
	capture := &sink.Capture{}

	if err := ProcessCore(context.Background(), cfg, testContext(), capture); err != nil {
		t.Fatalf("ProcessCore() error = %v", err)
	}

	msgs := srv.messages()
	if len(msgs) != 1 || msgs[0].subject != "cdc.users" || msgs[0].payload != "payload" {
		t.Fatalf("published = %+v, want one cdc.users message", msgs)
	}
	if !strings.Contains(msgs[0].header, "X-Table: users") || strings.Contains(msgs[0].header, "X-Empty") {
		t.Errorf("published header = %q, want X-Table only", msgs[0].header)
	}
	if !strings.Contains(srv.connectOptions()[0], `"name":"hump-test"`) {
		t.Errorf("CONNECT = %s, want name hump-test", srv.connectOptions()[0])
	}

	if capture.Target != "cdc.users" || string(capture.Payload) != "payload" {
		t.Errorf("capture = %q %q, want cdc.users payload", capture.Target, capture.Payload)
	}
	if got := capture.Metadata["header.X-Table"]; got != "users" {
		t.Errorf("capture header.X-Table = %q, want users", got)
	}
}

func TestProcessCoreDefaultsNameToHostname(t *testing.T) {
	srv := newFakeServer(t, nil)
	cfg := sinkcfg.NatsCoreSink{NatsCommon: sinkcfg.NatsCommon{
		URL:     mustTemplate(t, srv.url()),
		Subject: mustTemplate(t, "cdc"),
	}}
	forgetClients(t, srv.url())

	if err := ProcessCore(context.Background(), cfg, testContext(), nil); err != nil {
		t.Fatalf("ProcessCore() error = %v", err)
	}

	if err := ProcessCore(context.Background(), cfg, testContext(), nil); err != nil {
		t.Fatalf("ProcessCore() second error = %v", err)
	}
	if got := srv.clientCount(); got != 1 {
		t.Errorf("connections = %d, want 1 cached client", got)
	}
	if strings.Contains(srv.connectOptions()[0], `"name":""`) {
		t.Errorf("CONNECT = %s, want the hostname as name", srv.connectOptions()[0])
	}
}

func TestProcessCoreOnClosedConnectionIsTransient(t *testing.T) {
	srv := newFakeServer(t, nil)
	cfg := sinkcfg.NatsCoreSink{NatsCommon: sinkcfg.NatsCommon{
		URL:     mustTemplate(t, srv.url()),
		Subject: mustTemplate(t, "cdc"),
	}}
	forgetClients(t, srv.url())
	nc := mustClient(t, cfg.NatsCommon)
	nc.conn.Close()
	capture := &sink.Capture{}

	err := ProcessCore(context.Background(), cfg, testContext(), capture)
	if !errors.Is(err, nats.ErrConnectionClosed) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("ProcessCore() error = %v, want transient ErrConnectionClosed", err)
	}
	if capture.Target != "cdc" {
		t.Errorf("capture.Target = %q, want cdc", capture.Target)
	}
}

func TestProcessCoreErrors(t *testing.T) {
	bad := mustTemplate(t, "{{ .NoSuchField }}")
	srv := newFakeServer(t, nil)
	forgetClients(t, srv.url())
	url := mustTemplate(t, srv.url())

	tests := []struct {
		name      string
		cfg       sinkcfg.NatsCommon
		target    error
		permanent bool
	}{
		{"url", sinkcfg.NatsCommon{URL: bad}, template.ErrRender, true},
		{"name", sinkcfg.NatsCommon{URL: url, Name: bad}, template.ErrRender, true},
		{"auth", sinkcfg.NatsCommon{URL: url, Token: bad}, template.ErrRender, true},
		{"subject", sinkcfg.NatsCommon{URL: url, Subject: bad}, template.ErrRender, true},
		{"invalid subject", sinkcfg.NatsCommon{URL: url, Subject: mustTemplate(t, "a.*")}, errInvalidSubject, true},
		{"payload", sinkcfg.NatsCommon{URL: url, Subject: mustTemplate(t, "a"), Payload: bad}, template.ErrRender, true},
		{"header", sinkcfg.NatsCommon{
			URL:     url,
			Subject: mustTemplate(t, "a"),
			Headers: map[string]*template.Template{"X-Bad": bad},
		}, template.ErrRender, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProcessCore(context.Background(), sinkcfg.NatsCoreSink{NatsCommon: tt.cfg}, testContext(), nil)
			if !errors.Is(err, tt.target) {
				t.Fatalf("ProcessCore() error = %v, want %v", err, tt.target)
			}
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
				t.Errorf("permanent = %v, want %v", got, tt.permanent)
			}
		})
	}
}

func TestProcessReturnsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	common := sinkcfg.NatsCommon{URL: mustTemplate(t, "nats://127.0.0.1:1")}

	if err := ProcessCore(ctx, sinkcfg.NatsCoreSink{NatsCommon: common}, testContext(), nil); !errors.Is(err, context.Canceled) {
		t.Errorf("ProcessCore() error = %v, want context.Canceled", err)
	}
	if err := ProcessJetStream(ctx, sinkcfg.NatsJetStreamSink{NatsCommon: common}, testContext(), nil); !errors.Is(err, context.Canceled) {
		t.Errorf("ProcessJetStream() error = %v, want context.Canceled", err)
	}
}

func TestProcessJetStreamPublishesWithAck(t *testing.T) {
	srv := newFakeServer(t, func(string) []byte { return []byte(`{"stream":"orders","seq":7}`) })
	cfg := sinkcfg.NatsJetStreamSink{
		NatsCommon: sinkcfg.NatsCommon{
			URL:     mustTemplate(t, srv.url()),
			Subject: mustTemplate(t, "orders.{{ .Table }}"),
			Payload: mustTemplate(t, "payload"),
			Headers: map[string]*template.Template{
				headerNatsExpectedStream: mustTemplate(t, "spoofed"),
			},
		},
		Stream: sinkcfg.NatsStream{Name: "orders"},
		MsgID:  mustTemplate(t, "{{ .Table }}-1"),
	}
	forgetClients(t, srv.url())
	capture := &sink.Capture{}

	if err := ProcessJetStream(context.Background(), cfg, testContext(), capture); err != nil {
		t.Fatalf("ProcessJetStream() error = %v", err)
	}

	want := map[string]string{
		"stream":                             "orders",
		"sequence":                           "7",
		"header." + headerNatsMsgID:          "users-1",
		"header." + headerNatsExpectedStream: "orders",
	}
	for k, v := range want {
		if capture.Metadata[k] != v {
			t.Errorf("capture.Metadata[%q] = %q, want %q", k, capture.Metadata[k], v)
		}
	}
	if capture.Target != "orders.users" {
		t.Errorf("capture.Target = %q, want orders.users", capture.Target)
	}
}

func TestProcessJetStreamExpectedStreamMismatchIsTransient(t *testing.T) {
	srv := newFakeServer(t, func(string) []byte {
		return fmt.Appendf(nil, `{"error":{"code":400,"err_code":%d,"description":"expected stream does not match"}}`, errCodeExpectedStreamMismatch)
	})
	cfg := sinkcfg.NatsJetStreamSink{
		NatsCommon: sinkcfg.NatsCommon{URL: mustTemplate(t, srv.url()), Subject: mustTemplate(t, "orders")},
		Stream:     sinkcfg.NatsStream{Name: "orders"},
	}
	forgetClients(t, srv.url())
	capture := &sink.Capture{}

	err := ProcessJetStream(context.Background(), cfg, testContext(), capture)

	var apiErr *jetstream.APIError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != errCodeExpectedStreamMismatch {
		t.Fatalf("ProcessJetStream() error = %v, want API error %d", err, errCodeExpectedStreamMismatch)
	}
	if errors.Is(err, sink.ErrSinkPermanent) {
		t.Errorf("ProcessJetStream() error = %v, want transient", err)
	}
	if _, ok := capture.Metadata["sequence"]; ok {
		t.Error("capture has sequence, want none without an ack")
	}
	if capture.Metadata["stream"] != "orders" {
		t.Errorf("capture stream = %q, want orders", capture.Metadata["stream"])
	}
}

func TestProcessJetStreamErrors(t *testing.T) {
	bad := mustTemplate(t, "{{ .NoSuchField }}")
	srv := newFakeServer(t, nil)
	forgetClients(t, srv.url())
	url := mustTemplate(t, srv.url())

	nc := mustClient(t, sinkcfg.NatsCommon{URL: url})
	nc.reconciler = newReconciler(&fakeJS{createOrUpdateErr: errors.New("insufficient resources")})

	stream := sinkcfg.NatsStream{Name: "orders"}
	tests := []struct {
		name   string
		cfg    sinkcfg.NatsJetStreamSink
		target error
	}{
		{"url", sinkcfg.NatsJetStreamSink{NatsCommon: sinkcfg.NatsCommon{URL: bad}, Stream: stream}, template.ErrRender},
		{"reconcile", sinkcfg.NatsJetStreamSink{
			NatsCommon: sinkcfg.NatsCommon{URL: url},
			Stream:     sinkcfg.NatsStream{Name: "orders", Subjects: []string{"orders.>"}},
		}, errReconcile},
		{"subject", sinkcfg.NatsJetStreamSink{NatsCommon: sinkcfg.NatsCommon{URL: url, Subject: bad}, Stream: stream}, template.ErrRender},
		{"msgId", sinkcfg.NatsJetStreamSink{
			NatsCommon: sinkcfg.NatsCommon{URL: url, Subject: mustTemplate(t, "orders")},
			Stream:     stream,
			MsgID:      bad,
		}, template.ErrRender},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProcessJetStream(context.Background(), tt.cfg, testContext(), nil)
			if !errors.Is(err, tt.target) || !errors.Is(err, sink.ErrSinkPermanent) {
				t.Fatalf("ProcessJetStream() error = %v, want permanent %v", err, tt.target)
			}
		})
	}
}

func TestProcessJetStreamOnClosedConnectionIsTransient(t *testing.T) {
	srv := newFakeServer(t, nil)
	cfg := sinkcfg.NatsJetStreamSink{
		NatsCommon: sinkcfg.NatsCommon{URL: mustTemplate(t, srv.url()), Subject: mustTemplate(t, "orders")},
		Stream:     sinkcfg.NatsStream{Name: "orders"},
	}
	forgetClients(t, srv.url())
	mustClient(t, cfg.NatsCommon).conn.Close()

	err := ProcessJetStream(context.Background(), cfg, testContext(), nil)
	if !errors.Is(err, nats.ErrConnectionClosed) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("ProcessJetStream() error = %v, want transient ErrConnectionClosed", err)
	}
}

func TestGetClientRetriesUnreachableServer(t *testing.T) {
	url := "nats://" + closedAddr(t)
	forgetClients(t, url)

	nc := mustClient(t, sinkcfg.NatsCommon{URL: mustTemplate(t, url)})
	if nc.conn.IsConnected() {
		t.Error("IsConnected() = true, want a client still retrying the connect")
	}
}

func TestGetClientRejectsInvalidURL(t *testing.T) {
	_, err := getClient(sinkcfg.NatsCommon{URL: mustTemplate(t, "nats://%zz")}, testContext())
	if err == nil {
		t.Fatal("getClient() error = nil, want an invalid URL error")
	}
}

func TestGetClientConnectionErrorHidesSecretsAndPreservesCause(t *testing.T) {
	rawURL := diagnosticURL("nats", "127.0.0.1:invalid-port")
	nc, err := getClient(sinkcfg.NatsCommon{URL: mustTemplate(t, rawURL)}, testContext())
	if nc != nil || err == nil {
		t.Fatalf("getClient() = %v, %v, want no client and a URL error", nc, err)
	}
	if got, want := err.Error(), "nats client for [redacted endpoint]"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	var cause *url.Error
	if !errors.As(err, &cause) {
		t.Fatalf("getClient() error type = %T, want wrapped *url.Error", err)
	}
	if cause.URL != rawURL {
		t.Errorf("cause URL = %q, want original connection URL %q", cause.URL, rawURL)
	}
	if !errors.Is(err, cause) {
		t.Error("connection error does not preserve its cause")
	}
	for _, format := range []string{"%s", "%v", "%+v"} {
		text := fmt.Sprintf(format, err)
		for _, secret := range []string{"diagnostic-user-sentinel", "diagnostic-password-sentinel", "diagnostic-vhost-sentinel", "diagnostic-query-sentinel"} {
			if strings.Contains(text, secret) {
				t.Errorf("format %q exposed %q: %s", format, secret, text)
			}
		}
	}
}

func TestGetClientAuthenticates(t *testing.T) {
	user, err := nkeys.CreateUser()
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	seed, err := user.Seed()
	if err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	creds := fmt.Sprintf("-----BEGIN NATS USER JWT-----\neyJ0eXAiOiJKV1QifQ.e30.c2ln\n------END NATS USER JWT------\n\n"+
		"-----BEGIN USER NKEY SEED-----\n%s\n------END USER NKEY SEED------\n", seed)

	tests := []struct {
		name string
		cfg  sinkcfg.NatsCommon
		want []string
	}{
		{"credentials", sinkcfg.NatsCommon{Credentials: mustTemplate(t, creds)}, []string{`"jwt":"eyJ0eXAiOiJKV1QifQ.e30.c2ln"`, `"sig":`}},
		{"nkey seed", sinkcfg.NatsCommon{NKeySeed: mustTemplate(t, string(seed))}, []string{`"nkey":"U`, `"sig":`}},
		{"username only", sinkcfg.NatsCommon{Username: mustTemplate(t, "alice")}, []string{`"user":"alice"`}},
		{"password only", sinkcfg.NatsCommon{Password: mustTemplate(t, "secret")}, []string{`"pass":"secret"`}},
		{"token", sinkcfg.NatsCommon{Token: mustTemplate(t, "tok")}, []string{`"auth_token":"tok"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t, nil)
			forgetClients(t, srv.url())
			tt.cfg.URL = mustTemplate(t, srv.url())

			nc := mustClient(t, tt.cfg)
			if err := nc.conn.FlushTimeout(time.Second); err != nil {
				t.Fatalf("FlushTimeout() error = %v", err)
			}

			connect := srv.connectOptions()[0]
			for _, w := range tt.want {
				if !strings.Contains(connect, w) {
					t.Errorf("CONNECT = %s, want it to contain %s", connect, w)
				}
			}
		})
	}
}

func TestResolveAuthErrors(t *testing.T) {
	bad := mustTemplate(t, "{{ .NoSuchField }}")
	tests := []struct {
		name      string
		cfg       sinkcfg.NatsCommon
		permanent bool
	}{
		{"credentials render", sinkcfg.NatsCommon{Credentials: bad}, false},
		{"credentials without seed", sinkcfg.NatsCommon{Credentials: mustTemplate(t, "not a creds file")}, true},
		{"nkeySeed render", sinkcfg.NatsCommon{NKeySeed: bad}, false},
		{"nkeySeed invalid", sinkcfg.NatsCommon{NKeySeed: mustTemplate(t, "SUAINVALID")}, true},
		{"username render", sinkcfg.NatsCommon{Username: bad}, false},
		{"password render", sinkcfg.NatsCommon{Username: mustTemplate(t, "u"), Password: bad}, false},
		{"token render", sinkcfg.NatsCommon{Token: bad}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveAuth(tt.cfg, testContext())
			if err == nil {
				t.Fatal("resolveAuth() error = nil, want an error")
			}
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
				t.Errorf("resolveAuth() error = %v, permanent = %v, want %v", err, got, tt.permanent)
			}
			if !tt.permanent && !errors.Is(err, template.ErrRender) {
				t.Errorf("resolveAuth() error = %v, want a render error", err)
			}
		})
	}
}

func TestResolveAuthWithoutCredentials(t *testing.T) {
	auth, err := resolveAuth(sinkcfg.NatsCommon{Token: mustTemplate(t, "")}, testContext())
	if err != nil || auth.option != nil {
		t.Fatalf("resolveAuth() = %+v, %v, want no auth option", auth, err)
	}
}

func TestClientLogsReconnect(t *testing.T) {
	srv := newFakeServer(t, nil)
	forgetClients(t, srv.url())
	nc := mustClient(t, sinkcfg.NatsCommon{URL: mustTemplate(t, srv.url())})

	if err := nc.conn.ForceReconnect(); err != nil {
		t.Fatalf("ForceReconnect() error = %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for srv.clientCount() < 2 || !nc.conn.IsConnected() {
		if time.Now().After(deadline) {
			t.Fatalf("no reconnect: %d connections, status %v", srv.clientCount(), nc.conn.Status())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := nc.conn.Stats().Reconnects; got != 1 {
		t.Errorf("Reconnects = %d, want 1", got)
	}
}

func TestShutdownDrainsClients(t *testing.T) {
	if !runBrokerDiagnostics(t, "NATS asynchronous error", "NATS disconnected", "NATS reconnected", "NATS connection closed", "nats_sink(nats://127.0.0.1:", "Service stopped") {
		return
	}
	srv := newFakeServer(t, nil)
	url := diagnosticURL("nats", srv.ln.Addr().String())
	forgetClients(t, url)
	cfg := sinkcfg.NatsCoreSink{NatsCommon: sinkcfg.NatsCommon{
		URL: mustTemplate(t, url), Subject: mustTemplate(t, "cdc.users"), Payload: mustTemplate(t, "payload"),
	}}
	capture := &sink.Capture{}
	if err := ProcessCore(context.Background(), cfg, testContext(), capture); err != nil {
		t.Fatalf("ProcessCore() error = %v", err)
	}
	if capture.Target != "cdc.users" || string(capture.Payload) != "payload" || len(srv.messages()) != 1 {
		t.Fatal("diagnostic redaction changed publish or capture")
	}
	connect := srv.connectOptions()[0]
	if !strings.Contains(connect, `"user":"diagnostic-user-sentinel"`) || !strings.Contains(connect, `"pass":"diagnostic-password-sentinel"`) {
		t.Fatal("URL credentials did not reach CONNECT unchanged")
	}
	nc := mustClient(t, cfg.NatsCommon)
	nc.conn.Opts.AsyncErrorCB(nc.conn, nil, errors.New(url))
	nc.conn.Opts.DisconnectedErrCB(nc.conn, errors.New(url))
	reconnected := make(chan struct{})
	onReconnect := nc.conn.Opts.ReconnectedCB
	nc.conn.SetReconnectHandler(func(c *nats.Conn) {
		onReconnect(c)
		close(reconnected)
	})
	if err := nc.conn.ForceReconnect(); err != nil {
		t.Fatalf("ForceReconnect() error = %v", err)
	}
	select {
	case <-reconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect callback did not run")
	}
	closed := make(chan struct{})
	onClose := nc.conn.Opts.ClosedCB
	nc.conn.SetClosedHandler(func(c *nats.Conn) {
		onClose(c)
		close(closed)
	})

	_ = shutdown.Shutdown(time.Second)

	deadline := time.Now().Add(5 * time.Second)
	for !nc.conn.IsClosed() {
		if time.Now().After(deadline) {
			t.Fatalf("connection not closed after shutdown, status %v", nc.conn.Status())
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("closed callback did not run")
	}
}

func TestValidateSubject(t *testing.T) {
	tests := map[string]bool{
		"orders.created": true,
		"":               false,
		"a b":            false,
		"a.*":            false,
		"a.>":            false,
		"a..b":           false,
		".a":             false,
	}
	for subject, valid := range tests {
		err := validateSubject(subject)
		if (err == nil) != valid {
			t.Errorf("validateSubject(%q) = %v, want valid %v", subject, err, valid)
		}
		if err != nil && !errors.Is(err, errInvalidSubject) {
			t.Errorf("validateSubject(%q) = %v, want errInvalidSubject", subject, err)
		}
	}
}

func TestApplyJetStreamHeadersDropsEmptyMsgID(t *testing.T) {
	header := nats.Header{headerNatsMsgID: []string{"user-set"}}
	cfg := sinkcfg.NatsJetStreamSink{Stream: sinkcfg.NatsStream{Name: "orders"}, MsgID: mustTemplate(t, "")}

	if err := applyJetStreamHeaders(header, cfg, testContext()); err != nil {
		t.Fatalf("applyJetStreamHeaders() error = %v", err)
	}
	if _, ok := header[headerNatsMsgID]; ok {
		t.Errorf("header = %v, want %s removed", header, headerNatsMsgID)
	}
	if got := header.Get(headerNatsExpectedStream); got != "orders" {
		t.Errorf("%s = %q, want orders", headerNatsExpectedStream, got)
	}
}

func TestFlushTimeout(t *testing.T) {
	if got := flushTimeout(context.Background()); got != 5*time.Second {
		t.Errorf("flushTimeout(no deadline) = %v, want 5s", got)
	}

	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	t.Cleanup(cancel)
	if got := flushTimeout(expired); got != 5*time.Second {
		t.Errorf("flushTimeout(expired) = %v, want 5s", got)
	}

	bounded, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	if got := flushTimeout(bounded); got <= 0 || got > time.Minute {
		t.Errorf("flushTimeout(1m) = %v, want (0, 1m]", got)
	}
}

func TestWrapErrorClassifiesNatsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"render", fmt.Errorf("render: %w", template.ErrRender), true},
		{"invalid subject", errInvalidSubject, true},
		{"reconcile", errReconcile, true},
		{"max payload", nats.ErrMaxPayload, true},
		{"authorization", nats.ErrAuthorization, true},
		{"auth expired", nats.ErrAuthExpired, true},
		{"permission", nats.ErrPermissionViolation, true},
		{"stream not found", jetstream.ErrStreamNotFound, true},
		{"expected stream mismatch", &jetstream.APIError{ErrorCode: errCodeExpectedStreamMismatch}, false},
		{"other api error", &jetstream.APIError{ErrorCode: 10077}, false},
		{"timeout", nats.ErrTimeout, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(wrapError(tt.err), sink.ErrSinkPermanent); got != tt.want {
				t.Errorf("wrapError(%v) permanent = %v, want %v", tt.err, got, tt.want)
			}
		})
	}

	if isPermanentNatsError(nil) {
		t.Error("isPermanentNatsError(nil) = true, want false")
	}
}

func TestIsTransientConnError(t *testing.T) {
	transient := []error{
		nats.ErrNoServers, nats.ErrConnectionClosed, nats.ErrConnectionReconnecting,
		nats.ErrConnectionDraining, nats.ErrTimeout, context.DeadlineExceeded, context.Canceled,
	}
	for _, err := range transient {
		if !isTransientConnError(fmt.Errorf("wrapped: %w", err)) {
			t.Errorf("isTransientConnError(%v) = false, want true", err)
		}
	}

	for _, err := range []error{nil, errors.New("boom"), jetstream.ErrStreamNotFound} {
		if isTransientConnError(err) {
			t.Errorf("isTransientConnError(%v) = true, want false", err)
		}
	}
}

// runBrokerDiagnostics isolates logger initialization and the singleton shutdown registry.
func runBrokerDiagnostics(t *testing.T, messages ...string) bool {
	t.Helper()
	if os.Getenv("HUMP_BROKER_DIAGNOSTIC_CHILD") == "1" {
		return true
	}
	for _, level := range []string{"info", "debug"} {
		t.Run(level, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			t.Cleanup(cancel)
			args := []string{"-test.run=^TestShutdownDrainsClients$"}
			if dir := flag.Lookup("test.gocoverdir"); dir != nil && dir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+dir.Value.String())
			}
			cmd := exec.CommandContext(ctx, os.Args[0], args...)
			cmd.Env = append(os.Environ(), "HUMP_BROKER_DIAGNOSTIC_CHILD=1", "LOG_FORMAT=json", "LOG_VERBOSE="+strconv.FormatBool(level == "debug"))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("diagnostic child: %v\n%s", err, out)
			}
			text := string(out)
			for _, secret := range []string{"diagnostic-user-sentinel", "diagnostic-password-sentinel", "diagnostic-vhost-sentinel", "diagnostic-query-sentinel"} {
				if strings.Contains(text, secret) {
					t.Errorf("log exposed %q:\n%s", secret, text)
				}
			}
			for _, message := range messages {
				if !strings.Contains(text, message) {
					t.Errorf("log missing %q:\n%s", message, text)
				}
			}
			if got := strings.Contains(text, `"level":"DEBUG"`); got != (level == "debug") {
				t.Errorf("debug output = %v at %s level:\n%s", got, level, text)
			}
		})
	}
	return false
}

// diagnosticURL includes distinct sentinels for every private URL component.
func diagnosticURL(scheme, addr string) string {
	return scheme + "://diagnostic-user-sentinel:diagnostic-password-sentinel@" + addr + "/diagnostic-vhost-sentinel?token=diagnostic-query-sentinel"
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

// mustClient returns the cached client for cfg or fails the test.
func mustClient(t *testing.T, cfg sinkcfg.NatsCommon) *natsConn {
	t.Helper()

	nc, err := getClient(cfg, testContext())
	if err != nil {
		t.Fatalf("getClient() error = %v", err)
	}

	return nc
}

// forgetClients closes and evicts every cached client for url when the test
// ends, so tests do not share connections.
func forgetClients(t *testing.T, url string) {
	t.Helper()

	t.Cleanup(func() {
		clientsMu.Lock()
		defer clientsMu.Unlock()

		for key, c := range clients {
			if key.url == url {
				c.conn.Close()
				delete(clients, key)
			}
		}
	})
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

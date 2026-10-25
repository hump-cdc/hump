package rabbitmq_stream

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"

	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/template"
)

func TestProcessPublishesAndCaptures(t *testing.T) {
	srv := newFakeServer(t)
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModeCreateWhenMissing)
	cfg.Headers = map[string]*template.Template{
		"X-Table": mustTemplate(t, "{{ .Table }}"),
		"X-Empty": mustTemplate(t, ""),
	}
	forgetURL(t, srv.url())
	capture := &sink.Capture{}

	if err := Process(context.Background(), cfg, testContext(), capture); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if err := Process(context.Background(), cfg, testContext(), nil); err != nil {
		t.Fatalf("Process() second error = %v", err)
	}

	creates, published, declared := srv.snapshot()
	if !slices.Equal(creates, []string{"cdc-users"}) || declared != 1 {
		t.Errorf("creates, declared = %v, %d, want one create and one publisher", creates, declared)
	}
	if len(published) != 2 || !strings.Contains(published[0], "payload") || !strings.Contains(published[0], "X-Table") {
		t.Errorf("published = %q, want two messages carrying the payload and X-Table", published)
	}

	if capture.Target != "cdc-users" || string(capture.Payload) != "payload" {
		t.Errorf("capture = %q %q, want cdc-users payload", capture.Target, capture.Payload)
	}
	if capture.Metadata["header.X-Table"] != "users" {
		t.Errorf("capture.Metadata = %v, want header.X-Table users", capture.Metadata)
	}
}

func TestProcessPublishOutcomes(t *testing.T) {
	tests := []struct {
		name      string
		outcome   publishOutcome
		want      error
		permanent bool
	}{
		{"rejected", outcomeError, errPublishNacked, true},
		{"unanswered", outcomeSilent, context.DeadlineExceeded, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t)
			srv.set(func(s *fakeServer) { s.publish = func() publishOutcome { return tt.outcome } })
			forgetURL(t, srv.url())
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			t.Cleanup(cancel)
			if tt.outcome == outcomeSilent {
				ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
				t.Cleanup(cancel)
			}

			err := Process(ctx, streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModePassive), testContext(), nil)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Process() error = %v, want %v", err, tt.want)
			}
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
				t.Errorf("permanent = %v, want %v", got, tt.permanent)
			}
		})
	}
}

func TestProcessOnClosedProducerIsTransient(t *testing.T) {
	srv := newFakeServer(t)
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModePassive)
	forgetURL(t, srv.url())

	rp, err := getProducer(context.Background(), cfg, testContext(), "cdc-users")
	if err != nil {
		t.Fatalf("getProducer() error = %v", err)
	}
	key := producerKey{url: srv.url(), stream: "cdc-users"}
	_ = rp.prod.Close()
	waitEvicted(t, key)

	producersMu.Lock()
	producers[key] = rp
	producersMu.Unlock()

	err = Process(context.Background(), cfg, testContext(), &sink.Capture{})
	if err == nil || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want a transient send error", err)
	}
}

func TestProducerEvictedWhenConnectionDrops(t *testing.T) {
	srv := newFakeServer(t)
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModePassive)
	forgetURL(t, srv.url())

	if _, err := getProducer(context.Background(), cfg, testContext(), "cdc-users"); err != nil {
		t.Fatalf("getProducer() error = %v", err)
	}

	srv.dropClients()

	waitEvicted(t, producerKey{url: srv.url(), stream: "cdc-users"})
}

func TestProcessReconcilesStream(t *testing.T) {
	tests := []struct {
		name      string
		mode      sinkcfg.RabbitMQStreamMode
		metadata  uint16
		create    uint16
		want      error
		permanent bool
	}{
		{"passive missing", sinkcfg.RabbitMQStreamModePassive, codeStreamDoesNotExist, codeOK, errStreamMissing, true},
		{"create exists", sinkcfg.RabbitMQStreamModeCreateWhenMissing, codeOK, codeStreamExists, nil, false},
		{"create mismatch", sinkcfg.RabbitMQStreamModeCreateWhenMissing, codeOK, codePreconditionFailed, errStreamMismatch, true},
		{"create refused", sinkcfg.RabbitMQStreamModeCreateWhenMissing, codeOK, codeAccessRefused, stream.CodeAccessRefused, true},
		{"create unavailable", sinkcfg.RabbitMQStreamModeCreateWhenMissing, codeOK, codeStreamNotAvailable, stream.StreamNotAvailable, false},
		{"leader missing", sinkcfg.RabbitMQStreamModeCreateWhenMissing, codeStreamDoesNotExist, codeOK, stream.StreamDoesNotExist, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t)
			srv.set(func(s *fakeServer) {
				s.metadata = func(string) uint16 { return tt.metadata }
				s.create = func(string) uint16 { return tt.create }
			})
			forgetURL(t, srv.url())

			err := Process(context.Background(), streamSink(t, srv.url(), tt.mode), testContext(), nil)
			if tt.want == nil && err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("Process() error = %v, want %v", err, tt.want)
			}
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
				t.Errorf("permanent = %v, want %v (err %v)", got, tt.permanent, err)
			}
		})
	}
}

func TestEnsureCachesStickyOutcome(t *testing.T) {
	srv := newFakeServer(t)
	srv.set(func(s *fakeServer) { s.create = func(string) uint16 { return codePreconditionFailed } })
	forgetURL(t, srv.url())
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModeCreateWhenMissing)

	for range 2 {
		if err := Process(context.Background(), cfg, testContext(), nil); !errors.Is(err, errStreamMismatch) {
			t.Fatalf("Process() error = %v, want errStreamMismatch", err)
		}
	}

	if creates, _, _ := srv.snapshot(); len(creates) != 1 {
		t.Errorf("creates = %v, want one sticky declaration", creates)
	}
}

func TestEnsureRetriesTransientFailure(t *testing.T) {
	srv := newFakeServer(t)
	srv.set(func(s *fakeServer) { s.create = func(string) uint16 { return codeStreamNotAvailable } })
	forgetURL(t, srv.url())
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModeCreateWhenMissing)

	if err := Process(context.Background(), cfg, testContext(), nil); !errors.Is(err, stream.StreamNotAvailable) {
		t.Fatalf("Process() error = %v, want StreamNotAvailable", err)
	}

	srv.set(func(s *fakeServer) { s.create = func(string) uint16 { return codeOK } })
	if err := Process(context.Background(), cfg, testContext(), nil); err != nil {
		t.Fatalf("Process() retry error = %v", err)
	}
	if creates, _, _ := srv.snapshot(); len(creates) != 2 {
		t.Errorf("creates = %v, want the transient failure retried", creates)
	}
}

func TestGetProducerSharesConcurrentCreation(t *testing.T) {
	srv := newFakeServer(t)
	srv.set(func(s *fakeServer) { s.delay = 100 * time.Millisecond })
	forgetURL(t, srv.url())
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModeCreateWhenMissing)

	var (
		wg      sync.WaitGroup
		results [2]*rabbitProducer
		errs    [2]error
	)
	for i := range 2 {
		wg.Go(func() {
			time.Sleep(time.Duration(i) * 20 * time.Millisecond)
			results[i], errs[i] = getProducer(context.Background(), cfg, testContext(), "cdc-users")
		})
	}
	wg.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("getProducer() errors = %v", errs)
	}
	if results[0] != results[1] {
		t.Error("concurrent getProducer() created two producers")
	}
	if _, _, declared := srv.snapshot(); declared != 1 {
		t.Errorf("declared = %d, want 1", declared)
	}
}

func TestProcessErrors(t *testing.T) {
	bad := mustTemplate(t, "{{ .NoSuchField }}")
	with := func(set func(cfg *sinkcfg.RabbitMQStreamSink)) sinkcfg.RabbitMQStreamSink {
		cfg := streamSink(t, "rabbitmq-stream://guest:guest@127.0.0.1:1/", sinkcfg.RabbitMQStreamModePassive)
		set(&cfg)
		return cfg
	}
	unreachable := "rabbitmq-stream://guest:guest@" + closedAddr(t) + "/"

	tests := []struct {
		name      string
		cfg       sinkcfg.RabbitMQStreamSink
		permanent bool
	}{
		{"stream render", with(func(c *sinkcfg.RabbitMQStreamSink) { c.Stream.Name = bad }), true},
		{"payload render", with(func(c *sinkcfg.RabbitMQStreamSink) { c.Payload = bad }), true},
		{"header render", with(func(c *sinkcfg.RabbitMQStreamSink) { c.Headers = map[string]*template.Template{"X": bad} }), true},
		{"url render", with(func(c *sinkcfg.RabbitMQStreamSink) { c.URL = bad }), true},
		{"unreachable", with(func(c *sinkcfg.RabbitMQStreamSink) { c.URL = mustTemplate(t, unreachable) }), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Process(context.Background(), tt.cfg, testContext(), nil)
			if err == nil {
				t.Fatal("Process() error = nil, want an error")
			}
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.permanent {
				t.Errorf("Process() error = %v, permanent = %v, want %v", err, got, tt.permanent)
			}
		})
	}
}

func TestProcessReturnsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Process(ctx, streamSink(t, "rabbitmq-stream://127.0.0.1:1/", sinkcfg.RabbitMQStreamModePassive), testContext(), nil)
	if !errors.Is(err, context.Canceled) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want raw context.Canceled", err)
	}
}

func TestEnsureSkipsDeclareOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := newReconciler()

	err := r.ensure(ctx, nil, sinkcfg.RabbitMQStreamSink{}, "cdc-users")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ensure() error = %v, want context.Canceled", err)
	}
	if r.stateFor("cdc-users").done {
		t.Error("ensure() cached a cancelled declaration")
	}
}

func TestShutdownClosesProducersAndEnvironments(t *testing.T) {
	srv := newFakeServer(t)
	cfg := streamSink(t, srv.url(), sinkcfg.RabbitMQStreamModePassive)
	forgetURL(t, srv.url())

	rp, err := getProducer(context.Background(), cfg, testContext(), "cdc-users")
	if err != nil {
		t.Fatalf("getProducer() error = %v", err)
	}

	_ = shutdown.Shutdown(time.Second)

	if err := rp.prod.Send(amqp.NewMessage([]byte("late"))); err == nil {
		t.Error("Send() after shutdown error = nil, want a closed producer")
	}
	envsMu.Lock()
	_, cached := envs[envKey{url: srv.url()}]
	envsMu.Unlock()
	if cached {
		t.Error("environment still cached after shutdown")
	}
}

func TestToAMQPProps(t *testing.T) {
	if got := toAMQPProps(nil); got != nil {
		t.Errorf("toAMQPProps(nil) = %v, want nil", got)
	}

	got := toAMQPProps(domain.Headers{"a": "1", "b": ""})
	if len(got) != 1 || got["a"] != "1" {
		t.Errorf("toAMQPProps() = %v, want only a=1", got)
	}
}

func TestWrapErrorClassifiesStreamErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"render", template.ErrRender, true},
		{"missing", errStreamMissing, true},
		{"mismatch", errStreamMismatch, true},
		{"nacked", errPublishNacked, true},
		{"auth", stream.AuthenticationFailure, true},
		{"access refused", stream.CodeAccessRefused, true},
		{"precondition", stream.PreconditionFailed, true},
		{"connection closed", stream.ConnectionClosed, false},
		{"plain", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(wrapError(tt.err), sink.ErrSinkPermanent); got != tt.want {
				t.Errorf("wrapError(%v) permanent = %v, want %v", tt.err, got, tt.want)
			}
		})
	}

	if wrapError(nil) != nil {
		t.Error("wrapError(nil) != nil")
	}
	if isPermanentStreamError(nil) {
		t.Error("isPermanentStreamError(nil) = true, want false")
	}
}

func TestIsTransientStreamError(t *testing.T) {
	transient := []error{stream.ConnectionClosed, stream.AlreadyClosed, stream.StreamNotAvailable, context.DeadlineExceeded, context.Canceled}
	for _, err := range transient {
		if !isTransientStreamError(err) {
			t.Errorf("isTransientStreamError(%v) = false, want true", err)
		}
	}

	for _, err := range []error{nil, errors.New("boom"), errStreamMissing} {
		if isTransientStreamError(err) {
			t.Errorf("isTransientStreamError(%v) = true, want false", err)
		}
	}
}

func TestJitter(t *testing.T) {
	if got := jitter(time.Second, time.Second); got != time.Second {
		t.Errorf("jitter(1s, 1s) = %v, want 1s", got)
	}
	if got := jitter(time.Millisecond, 2*time.Millisecond); got < time.Millisecond || got > 2*time.Millisecond {
		t.Errorf("jitter(1ms, 2ms) = %v, want within [1ms, 2ms]", got)
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

// streamSink returns a sink publishing "payload" to stream "cdc-<table>" on
// url, reconciled in mode.
func streamSink(t *testing.T, url string, mode sinkcfg.RabbitMQStreamMode) sinkcfg.RabbitMQStreamSink {
	t.Helper()

	return sinkcfg.RabbitMQStreamSink{
		RabbitMQCommon: sinkcfg.RabbitMQCommon{URL: mustTemplate(t, url), Payload: mustTemplate(t, "payload")},
		Stream:         sinkcfg.RabbitMQStreamConfig{Name: mustTemplate(t, "cdc-{{ .Table }}"), Mode: mode},
	}
}

// waitEvicted waits until the producer cache no longer holds key.
func waitEvicted(t *testing.T, key producerKey) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		producersMu.Lock()
		_, cached := producers[key]
		producersMu.Unlock()

		if !cached {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("producer %v still cached", key)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// forgetURL closes and evicts every cached producer and environment for url
// when the test ends, so tests do not share connections.
func forgetURL(t *testing.T, url string) {
	t.Helper()

	t.Cleanup(func() {
		producersMu.Lock()
		for key, p := range producers {
			if key.url == url {
				_ = p.prod.Close()
				delete(producers, key)
			}
		}
		producersMu.Unlock()

		envsMu.Lock()
		if e, ok := envs[envKey{url: url}]; ok {
			_ = e.env.Close()
			delete(envs, envKey{url: url})
		}
		envsMu.Unlock()
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

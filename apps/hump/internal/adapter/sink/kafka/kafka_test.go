package kafka

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/template"
)

func TestProcessReturnsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Process(ctx, sinkcfg.KafkaSink{URL: mustTemplate(t, "127.0.0.1:1")}, testContext(), nil)
	if !errors.Is(err, context.Canceled) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want raw context.Canceled", err)
	}
}

func TestProcessCapturesRecordOnProduceFailure(t *testing.T) {
	cfg := sinkcfg.KafkaSink{
		URL:          mustTemplate(t, closedAddr(t)),
		Topic:        mustTemplate(t, "{{ .Schema }}.{{ .Table }}"),
		PartitionKey: mustTemplate(t, "key-{{ .Table }}"),
		Payload:      mustTemplate(t, "payload"),
		Headers: map[string]*template.Template{
			"X-Table": mustTemplate(t, "{{ .Table }}"),
			"X-Empty": mustTemplate(t, ""),
		},
	}
	closedClient(t, cfg)
	capture := &sink.Capture{}

	err := Process(context.Background(), cfg, testContext(), capture)
	if !errors.Is(err, kgo.ErrClientClosed) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want transient ErrClientClosed", err)
	}

	if capture.Target != "public.users" || string(capture.Payload) != "payload" {
		t.Errorf("capture = %q %q, want public.users payload", capture.Target, capture.Payload)
	}
	want := map[string]string{"key": "key-users", "header.X-Table": "users"}
	if len(capture.Metadata) != len(want) {
		t.Errorf("capture.Metadata = %v, want %v", capture.Metadata, want)
	}
	for k, v := range want {
		if capture.Metadata[k] != v {
			t.Errorf("capture.Metadata[%q] = %q, want %q", k, capture.Metadata[k], v)
		}
	}
}

func TestProcessWithoutCaptureAndKey(t *testing.T) {
	cfg := sinkcfg.KafkaSink{URL: mustTemplate(t, closedAddr(t)), Topic: mustTemplate(t, "t")}
	closedClient(t, cfg)
	capture := &sink.Capture{}

	if err := Process(context.Background(), cfg, testContext(), nil); !errors.Is(err, kgo.ErrClientClosed) {
		t.Fatalf("Process(nil capture) error = %v, want ErrClientClosed", err)
	}
	if err := Process(context.Background(), cfg, testContext(), capture); !errors.Is(err, kgo.ErrClientClosed) {
		t.Fatalf("Process() error = %v, want ErrClientClosed", err)
	}
	if _, ok := capture.Metadata["key"]; ok {
		t.Error("capture has key, want none for an empty partition key")
	}
}

func TestProcessProduceTimesOutAgainstUnreachableBroker(t *testing.T) {
	cfg := sinkcfg.KafkaSink{URL: mustTemplate(t, closedAddr(t)), Topic: mustTemplate(t, "t")}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	t.Cleanup(cancel)
	t.Cleanup(func() { forgetClients(t, cfg) })

	err := Process(ctx, cfg, testContext(), nil)
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want transient deadline exceeded", err)
	}
}

func TestProcessRenderFailuresArePermanent(t *testing.T) {
	bad := mustTemplate(t, "{{ .NoSuchField }}")
	tests := []struct {
		name string
		cfg  sinkcfg.KafkaSink
	}{
		{"url", sinkcfg.KafkaSink{URL: bad}},
		{"username", sinkcfg.KafkaSink{URL: mustTemplate(t, "h:1"), Username: bad}},
		{"password", sinkcfg.KafkaSink{URL: mustTemplate(t, "h:1"), Password: bad}},
		{"clientId", sinkcfg.KafkaSink{URL: mustTemplate(t, "h:1"), ClientID: bad}},
		{"topic", sinkcfg.KafkaSink{URL: mustTemplate(t, "render-topic:1"), Topic: bad}},
		{"partitionKey", sinkcfg.KafkaSink{URL: mustTemplate(t, "render-key:1"), PartitionKey: bad}},
		{"payload", sinkcfg.KafkaSink{URL: mustTemplate(t, "render-payload:1"), Payload: bad}},
		{"header", sinkcfg.KafkaSink{
			URL:     mustTemplate(t, "render-header:1"),
			Headers: map[string]*template.Template{"X-Bad": bad},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() { forgetClients(t, tt.cfg) })

			err := Process(context.Background(), tt.cfg, testContext(), nil)
			if !errors.Is(err, template.ErrRender) || !errors.Is(err, sink.ErrSinkPermanent) {
				t.Fatalf("Process() error = %v, want permanent render error", err)
			}
		})
	}
}

func TestGetClientCachesByRenderedSettings(t *testing.T) {
	cfg := sinkcfg.KafkaSink{
		URL:      mustTemplate(t, "a.invalid:9092, b.invalid:9092"),
		ClientID: mustTemplate(t, "hump-test"),
		TLS:      true,
	}
	t.Cleanup(func() { forgetClients(t, cfg) })

	first, err := getClient(cfg, testContext())
	if err != nil {
		t.Fatalf("getClient() error = %v", err)
	}
	second, err := getClient(cfg, testContext())
	if err != nil {
		t.Fatalf("getClient() second error = %v", err)
	}
	if first != second {
		t.Error("getClient() returned a new client, want the cached one")
	}

	other := cfg
	other.ClientID = mustTemplate(t, "hump-other")
	t.Cleanup(func() { forgetClients(t, other) })

	third, err := getClient(other, testContext())
	if err != nil {
		t.Fatalf("getClient(other) error = %v", err)
	}
	if third == first {
		t.Error("getClient() with another clientId returned the cached client")
	}
}

func TestGetClientSASL(t *testing.T) {
	for _, mechanism := range []string{"plain", "scram-sha-256", "scram-sha-512"} {
		t.Run(mechanism, func(t *testing.T) {
			cfg := sinkcfg.KafkaSink{
				URL:      mustTemplate(t, "sasl.invalid:9092"),
				SASL:     mechanism,
				Username: mustTemplate(t, "user"),
				Password: mustTemplate(t, "pass"),
			}
			t.Cleanup(func() { forgetClients(t, cfg) })

			if _, err := getClient(cfg, testContext()); err != nil {
				t.Fatalf("getClient() error = %v", err)
			}
		})
	}
}

func TestGetClientRejectsUnknownSASL(t *testing.T) {
	cfg := sinkcfg.KafkaSink{URL: mustTemplate(t, "sasl.invalid:9092"), SASL: "gssapi"}

	if _, err := getClient(cfg, testContext()); err == nil {
		t.Fatal("getClient() error = nil, want unsupported sasl mechanism")
	}
}

func TestGetClientRejectsInvalidSeed(t *testing.T) {
	cfg := sinkcfg.KafkaSink{URL: mustTemplate(t, "127.0.0.1:notaport")}

	if _, err := getClient(cfg, testContext()); err == nil {
		t.Fatal("getClient() error = nil, want invalid seed broker")
	}
}

func TestShutdownClosesClients(t *testing.T) {
	cfg := sinkcfg.KafkaSink{URL: mustTemplate(t, "shutdown.invalid:9092")}
	t.Cleanup(func() { forgetClients(t, cfg) })

	client, err := getClient(cfg, testContext())
	if err != nil {
		t.Fatalf("getClient() error = %v", err)
	}

	if err := shutdown.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if err := client.ProduceSync(context.Background(), &kgo.Record{Topic: "t"}).FirstErr(); !errors.Is(err, kgo.ErrClientClosed) {
		t.Errorf("ProduceSync() after shutdown error = %v, want ErrClientClosed", err)
	}
}

func TestWrapErrorClassifiesKafkaErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"unknown topic", fmt.Errorf("produce: %w", kerr.UnknownTopicOrPartition), true},
		{"other kafka error", kerr.NotLeaderForPartition, false},
		{"plain error", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapError(tt.err)
			if got := errors.Is(err, sink.ErrSinkPermanent); got != tt.want {
				t.Errorf("wrapError(%v) permanent = %v, want %v", tt.err, got, tt.want)
			}
			if !errors.Is(err, tt.err) {
				t.Errorf("wrapError(%v) = %v, want it to wrap the cause", tt.err, err)
			}
		})
	}
}

func TestConnHooksTrackBrokerState(t *testing.T) {
	h := newConnHooks("seed:9092")
	broker := kgo.BrokerMetadata{NodeID: 1, Host: "b1", Port: 9092}
	dialErr := errors.New("refused")

	steps := []struct {
		err  error
		want brokerState
	}{
		{nil, stateConnected},
		{nil, stateConnected},
		{dialErr, stateFailed},
		{dialErr, stateFailed},
		{nil, stateConnected},
	}
	for i, step := range steps {
		h.OnBrokerConnect(broker, 0, nil, step.err)
		if got := h.state[broker.NodeID]; got != step.want {
			t.Errorf("step %d: state = %v, want %v", i, got, step.want)
		}
	}

	h.OnBrokerDisconnect(broker, nil)
	if got := h.state[broker.NodeID]; got != stateConnected {
		t.Errorf("state after disconnect = %v, want unchanged %v", got, stateConnected)
	}

	h.OnClientClosed(nil)
}

func TestBrokerString(t *testing.T) {
	tests := []struct {
		meta kgo.BrokerMetadata
		want string
	}{
		{kgo.BrokerMetadata{NodeID: 3, Host: "b3", Port: 9092}, "b3:9092"},
		{kgo.BrokerMetadata{NodeID: -2147483648, Host: "s", Port: 9093}, "seed:s:9093"},
	}
	for _, tt := range tests {
		if got := brokerString(tt.meta); got != tt.want {
			t.Errorf("brokerString(%+v) = %q, want %q", tt.meta, got, tt.want)
		}
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

// closedClient caches a closed client for cfg, so Process reuses it and its
// produce fails at once with kgo.ErrClientClosed.
func closedClient(t *testing.T, cfg sinkcfg.KafkaSink) {
	t.Helper()

	client, err := getClient(cfg, testContext())
	if err != nil {
		t.Fatalf("getClient() error = %v", err)
	}
	client.Close()
	t.Cleanup(func() { forgetClients(t, cfg) })
}

// forgetClients closes and evicts the cached client cfg renders to, if any.
func forgetClients(t *testing.T, cfg sinkcfg.KafkaSink) {
	t.Helper()

	render := func(tmpl *template.Template) string {
		s, _ := tmpl.Render(testContext())
		return s
	}
	key := clientKey{
		url:      render(cfg.URL),
		clientID: render(cfg.ClientID),
		username: render(cfg.Username),
		password: render(cfg.Password),
		sasl:     cfg.SASL,
		tls:      cfg.TLS,
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[key]; ok {
		client.Close()
		delete(clients, key)
	}
}

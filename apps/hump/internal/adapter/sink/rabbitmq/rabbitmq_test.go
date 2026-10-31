package rabbitmq

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/template"
)

func TestProcessPublishesAndCaptures(t *testing.T) {
	srv := newFakeServer(t)
	cfg := rabbitSink(t, srv.url())
	cfg.Headers = map[string]*template.Template{
		"X-Table": mustTemplate(t, "{{ .Table }}"),
		"X-Empty": mustTemplate(t, ""),
	}
	forgetClients(t, srv.url())
	capture := &sink.Capture{}

	if err := Process(context.Background(), cfg, testContext(), capture); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if err := Process(context.Background(), cfg, testContext(), nil); err != nil {
		t.Fatalf("Process() second error = %v", err)
	}

	if got, want := srv.publishes(), []string{"events/public.users:payload", "events/public.users:payload"}; !slices.Equal(got, want) {
		t.Errorf("published = %v, want %v", got, want)
	}
	if got, want := srv.declared(), []string{"passive:events"}; !slices.Equal(got, want) {
		t.Errorf("declared = %v, want %v (one sticky reconcile)", got, want)
	}

	if capture.Target != "events" || string(capture.Payload) != "payload" {
		t.Errorf("capture = %q %q, want events payload", capture.Target, capture.Payload)
	}
	want := map[string]string{"routingKey": "public.users", "header.X-Table": "users"}
	if len(capture.Metadata) != len(want) {
		t.Errorf("capture.Metadata = %v, want %v", capture.Metadata, want)
	}
	for k, v := range want {
		if capture.Metadata[k] != v {
			t.Errorf("capture.Metadata[%q] = %q, want %q", k, capture.Metadata[k], v)
		}
	}
}

func TestProcessPublishOutcomes(t *testing.T) {
	tests := []struct {
		name    string
		outcome publishOutcome
		want    error
	}{
		{"nack", outcomeNack, errPublishNacked},
		{"unroutable", outcomeReturn, errUnroutable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t)
			srv.set(func(s *fakeServer) { s.publish = func(string, string) publishOutcome { return tt.outcome } })
			forgetClients(t, srv.url())

			err := Process(context.Background(), rabbitSink(t, srv.url()), testContext(), nil)
			if !errors.Is(err, tt.want) || !errors.Is(err, sink.ErrSinkPermanent) {
				t.Fatalf("Process() error = %v, want permanent %v", err, tt.want)
			}
		})
	}
}

func TestProcessReconcilesExchange(t *testing.T) {
	tests := []struct {
		name     string
		mode     sinkcfg.RabbitMQExchangeMode
		durable  bool
		declare  func(string, bool) uint16
		want     error
		declared []string
	}{
		{
			name:     "passive found",
			mode:     sinkcfg.RabbitMQExchangeModePassive,
			durable:  true,
			declare:  func(string, bool) uint16 { return 0 },
			declared: []string{"passive:events"},
		},
		{
			name:     "passive missing",
			mode:     sinkcfg.RabbitMQExchangeModePassive,
			declare:  func(string, bool) uint16 { return amqp.NotFound },
			want:     errExchangeMissing,
			declared: []string{"passive:events"},
		},
		{
			name:     "create when missing declares",
			mode:     sinkcfg.RabbitMQExchangeModeCreateWhenMissing,
			declare:  notFoundWhenPassive(0),
			declared: []string{"passive:events", "declare:events"},
		},
		{
			name:     "create when missing declare mismatch",
			mode:     sinkcfg.RabbitMQExchangeModeCreateWhenMissing,
			declare:  notFoundWhenPassive(amqp.PreconditionFailed),
			want:     errExchangeMismatch,
			declared: []string{"passive:events", "declare:events"},
		},
		{
			name:     "create when missing probe mismatch",
			mode:     sinkcfg.RabbitMQExchangeModeCreateWhenMissing,
			declare:  func(string, bool) uint16 { return amqp.PreconditionFailed },
			want:     errExchangeMismatch,
			declared: []string{"passive:events"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t)
			srv.set(func(s *fakeServer) { s.declare = tt.declare })
			forgetClients(t, srv.url())
			cfg := rabbitSink(t, srv.url())
			cfg.Exchange.Mode = tt.mode
			cfg.Exchange.Durable = &tt.durable

			err := Process(context.Background(), cfg, testContext(), nil)
			if tt.want == nil && err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if tt.want != nil && (!errors.Is(err, tt.want) || !errors.Is(err, sink.ErrSinkPermanent)) {
				t.Fatalf("Process() error = %v, want permanent %v", err, tt.want)
			}
			if got := srv.declared(); !slices.Equal(got, tt.declared) {
				t.Errorf("declared = %v, want %v", got, tt.declared)
			}
		})
	}
}

func TestProcessReconcileReopenFailureIsTransient(t *testing.T) {
	srv := newFakeServer(t)
	srv.set(func(s *fakeServer) {
		s.declare = func(_ string, passive bool) uint16 {
			srv.set(func(s *fakeServer) { s.failChannel = true })
			if passive {
				return amqp.NotFound
			}
			return 0
		}
	})
	forgetClients(t, srv.url())

	err := Process(context.Background(), rabbitSink(t, srv.url()), testContext(), nil)

	var amqpErr *amqp.Error
	if !errors.As(err, &amqpErr) || amqpErr.Code != amqp.ChannelError || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want transient channel error", err)
	}
}

func TestProcessWaitsForReadyConnection(t *testing.T) {
	url := "amqp://not-ready.invalid/"
	rc := &rabbitConn{url: url, ready: make(chan struct{}), quit: make(chan struct{})}
	injectClient(t, rc)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	t.Cleanup(cancel)

	err := Process(ctx, rabbitSink(t, url), testContext(), nil)
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want transient deadline exceeded", err)
	}
}

func TestProcessWithoutConnectionIsReconnecting(t *testing.T) {
	url := "amqp://reconnecting.invalid/"
	rc := &rabbitConn{url: url, ready: make(chan struct{}), quit: make(chan struct{})}
	rc.reconciler = newExchangeReconciler(rc.healthyChannel)
	rc.signalReady()
	injectClient(t, rc)

	err := Process(context.Background(), rabbitSink(t, url), testContext(), nil)
	if !errors.Is(err, errReconnecting) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want transient errReconnecting", err)
	}

	rc.reconciler.stateFor("events").done = true
	if err := Process(context.Background(), rabbitSink(t, url), testContext(), nil); !errors.Is(err, errReconnecting) {
		t.Fatalf("Process() with a reconciled exchange error = %v, want errReconnecting", err)
	}
}

func TestProcessReturnsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Process(ctx, rabbitSink(t, "amqp://127.0.0.1:1/"), testContext(), nil)
	if !errors.Is(err, context.Canceled) || errors.Is(err, sink.ErrSinkPermanent) {
		t.Fatalf("Process() error = %v, want raw context.Canceled", err)
	}
}

func TestProcessErrors(t *testing.T) {
	srv := newFakeServer(t)
	forgetClients(t, srv.url())
	bad := mustTemplate(t, "{{ .NoSuchField }}")

	badURL := rabbitSink(t, srv.url())
	badURL.URL = bad
	badExchange := rabbitSink(t, srv.url())
	badExchange.Exchange.Name = bad

	tests := []struct {
		name      string
		cfg       sinkcfg.RabbitMQSink
		permanent bool
	}{
		{"url render", badURL, true},
		{"exchange render", badExchange, true},
		{"dial", rabbitSink(t, "amqp://"+closedAddr(t)+"/"), false},
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

func TestGetClientFailsToOpenChannel(t *testing.T) {
	tests := []struct {
		name string
		set  func(s *fakeServer)
	}{
		{"channel", func(s *fakeServer) { s.failChannel = true }},
		{"confirm", func(s *fakeServer) { s.failConfirm = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeServer(t)
			srv.set(tt.set)

			if _, err := getClient(rabbitSink(t, srv.url()), testContext()); err == nil {
				t.Fatal("getClient() error = nil, want an open error")
			}
			if _, cached := clients[clientKey{url: srv.url()}]; cached {
				t.Error("getClient() cached a failed client")
			}
		})
	}
}

func TestHealthyChannelReopensClosedChannel(t *testing.T) {
	srv := newFakeServer(t)
	forgetClients(t, srv.url())
	rc := mustClient(t, srv.url())

	first, err := rc.healthyChannel()
	if err != nil || first != rc.ch {
		t.Fatalf("healthyChannel() = %v, %v, want the open channel", first, err)
	}
	_ = first.Close()

	srv.set(func(s *fakeServer) { s.failChannel = true })
	if _, err := rc.healthyChannel(); err == nil {
		t.Fatal("healthyChannel() error = nil, want channel.open refused")
	}

	srv.set(func(s *fakeServer) { s.failChannel, s.failConfirm = false, true })
	if _, err := rc.healthyChannel(); err == nil {
		t.Fatal("healthyChannel() error = nil, want confirm.select refused")
	}

	srv.set(func(s *fakeServer) { s.failConfirm = false })
	reopened, err := rc.healthyChannel()
	if err != nil {
		t.Fatalf("healthyChannel() error = %v", err)
	}
	if reopened == first || reopened.IsClosed() {
		t.Error("healthyChannel() did not reopen the channel")
	}
}

func TestShutdownClosesClients(t *testing.T) {
	if !runBrokerDiagnostics(t, "RabbitMQ connection established", "RabbitMQ channel reopened", "RabbitMQ connection closed", "RabbitMQ reconnect failed", "RabbitMQ reconnected", "rabbitmq_sink(amqp://127.0.0.1:", "Service stopped") {
		return
	}
	srv := newFakeServer(t)
	url := diagnosticURL(srv.ln.Addr().String())
	forgetClients(t, url)
	capture := &sink.Capture{}
	if err := Process(context.Background(), rabbitSink(t, url), testContext(), capture); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if capture.Target != "events" || string(capture.Payload) != "payload" || capture.Metadata["routingKey"] != "public.users" || len(srv.publishes()) != 1 {
		t.Fatal("diagnostic redaction changed publish or capture")
	}
	srv.mu.Lock()
	handshake := strings.Join(srv.handshakes, " ")
	srv.mu.Unlock()
	for _, value := range []string{"diagnostic-user-sentinel", "diagnostic-password-sentinel", "diagnostic-vhost-sentinel"} {
		if !strings.Contains(handshake, value) {
			t.Fatalf("connection handshake did not preserve %s", value)
		}
	}
	rc := mustClient(t, url)
	if rc.url != url {
		t.Fatal("cached connection URL changed")
	}
	_ = rc.ch.Close()
	if _, err := rc.healthyChannel(); err != nil {
		t.Fatalf("healthyChannel() error = %v", err)
	}
	srv.set(func(s *fakeServer) { s.rejectConnect = true })
	srv.dropClients()
	deadline := time.Now().Add(10 * time.Second)
	for srv.flag(func(s *fakeServer) bool { return s.rejected == 0 }) {
		if time.Now().After(deadline) {
			t.Fatal("supervisor did not attempt reconnect")
		}
		time.Sleep(10 * time.Millisecond)
	}
	srv.set(func(s *fakeServer) { s.rejectConnect = false })
	for !isReconnected(rc) {
		if time.Now().After(deadline) {
			t.Fatal("supervisor did not reconnect")
		}
		time.Sleep(10 * time.Millisecond)
	}
	rc.mu.Lock()
	conn := rc.conn
	rc.mu.Unlock()

	idle := newFakeServer(t)
	forgetClients(t, idle.url())
	stopped := mustClient(t, idle.url())
	close(stopped.quit)

	if err := shutdown.Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if !conn.IsClosed() {
		t.Error("connection open after shutdown")
	}
	if rc.conn != nil || rc.ch != nil {
		t.Error("shutdown left the connection on the client")
	}
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if len(clients) != 0 {
		t.Errorf("clients = %v, want every client evicted", clients)
	}
}

func TestSuperviseStopsOnClosedNotify(t *testing.T) {
	notify := make(chan *amqp.Error)
	close(notify)
	rc := &rabbitConn{closeNotif: notify, quit: make(chan struct{})}

	done := make(chan struct{})
	go func() {
		rc.supervise()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("supervise() did not return on a closed notify channel")
	}
}

func TestSuperviseStopsDuringBackoff(t *testing.T) {
	notify := make(chan *amqp.Error, 1)
	notify <- amqp.ErrClosed
	rc := &rabbitConn{closeNotif: notify, quit: make(chan struct{}), ready: make(chan struct{})}
	rc.reconciler = newExchangeReconciler(rc.healthyChannel)
	close(rc.ready)

	done := make(chan struct{})
	go func() {
		rc.supervise()
		close(done)
	}()

	deadline := time.Now().Add(time.Second)
	for isClosed(rc.readyChan()) {
		if time.Now().After(deadline) {
			t.Fatal("supervise() did not invalidate ready")
		}
		time.Sleep(time.Millisecond)
	}
	close(rc.quit)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("supervise() did not stop on quit")
	}
}

func TestSuperviseReconnects(t *testing.T) {
	t.Parallel()

	srv := newFakeServer(t)
	forgetClients(t, srv.url())
	rc := mustClient(t, srv.url())
	rc.reconciler.stateFor("events").done = true

	srv.dropClients()

	deadline := time.Now().Add(10 * time.Second)
	for isClosed(rc.readyChan()) {
		if time.Now().After(deadline) {
			t.Fatal("supervise() did not notice the dropped connection")
		}
		time.Sleep(time.Millisecond)
	}
	for !isReconnected(rc) {
		if time.Now().After(deadline) {
			t.Fatal("supervise() did not reconnect")
		}
		time.Sleep(10 * time.Millisecond)
	}

	rc.reconciler.mu.Lock()
	defer rc.reconciler.mu.Unlock()
	if len(rc.reconciler.states) != 0 {
		t.Error("reconnect kept the reconcile state, want it invalidated")
	}
}

func TestSuperviseRetriesFailedReconnect(t *testing.T) {
	t.Parallel()

	srv := newFakeServer(t)
	forgetClients(t, srv.url())
	rc := mustClient(t, srv.url())

	srv.set(func(s *fakeServer) { s.rejectConnect = true })
	srv.dropClients()

	deadline := time.Now().Add(10 * time.Second)
	for srv.flag(func(s *fakeServer) bool { return s.rejected == 0 }) {
		if time.Now().After(deadline) {
			t.Fatal("supervise() did not retry the connection")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if isClosed(rc.readyChan()) {
		t.Error("ready signalled after a failed reconnect")
	}
}

func TestBuildMessageRendersProperties(t *testing.T) {
	mandatory := false
	cfg := rabbitSink(t, "amqp://x/")
	cfg.Mandatory = &mandatory
	cfg.DeliveryMode = sinkcfg.RabbitMQDeliveryTransient
	cfg.Properties = sinkcfg.RabbitMQProperties{
		MessageID:       mustTemplate(t, "id"),
		CorrelationID:   mustTemplate(t, "corr"),
		Type:            mustTemplate(t, "type"),
		AppID:           mustTemplate(t, "app"),
		UserID:          mustTemplate(t, "user"),
		ContentType:     mustTemplate(t, "application/json"),
		ContentEncoding: mustTemplate(t, "gzip"),
		Expiration:      mustTemplate(t, "60000"),
		Priority:        mustTemplate(t, "9"),
	}

	msg, err := buildMessage(cfg, testContext())
	if err != nil {
		t.Fatalf("buildMessage() error = %v", err)
	}

	p := msg.Publishing
	got := []string{p.MessageId, p.CorrelationId, p.Type, p.AppId, p.UserId, p.ContentType, p.ContentEncoding, p.Expiration}
	want := []string{"id", "corr", "type", "app", "user", "application/json", "gzip", "60000"}
	if !slices.Equal(got, want) {
		t.Errorf("properties = %v, want %v", got, want)
	}
	if p.Priority != 9 || p.DeliveryMode != amqp.Transient || msg.Mandatory || p.Headers != nil {
		t.Errorf("priority, mode, mandatory, headers = %d, %d, %v, %v, want 9, transient, false, nil",
			p.Priority, p.DeliveryMode, msg.Mandatory, p.Headers)
	}
}

func TestBuildMessageErrors(t *testing.T) {
	bad := mustTemplate(t, "{{ .NoSuchField }}")
	props := func(set func(p *sinkcfg.RabbitMQProperties)) sinkcfg.RabbitMQSink {
		cfg := rabbitSink(t, "amqp://x/")
		set(&cfg.Properties)
		return cfg
	}
	with := func(set func(cfg *sinkcfg.RabbitMQSink)) sinkcfg.RabbitMQSink {
		cfg := rabbitSink(t, "amqp://x/")
		set(&cfg)
		return cfg
	}

	tests := []struct {
		name   string
		cfg    sinkcfg.RabbitMQSink
		render bool
	}{
		{"routingKey", with(func(c *sinkcfg.RabbitMQSink) { c.RoutingKey = bad }), true},
		{"payload", with(func(c *sinkcfg.RabbitMQSink) { c.Payload = bad }), true},
		{"header", with(func(c *sinkcfg.RabbitMQSink) { c.Headers = map[string]*template.Template{"X": bad} }), true},
		{"messageId", props(func(p *sinkcfg.RabbitMQProperties) { p.MessageID = bad }), true},
		{"correlationId", props(func(p *sinkcfg.RabbitMQProperties) { p.CorrelationID = bad }), true},
		{"type", props(func(p *sinkcfg.RabbitMQProperties) { p.Type = bad }), true},
		{"appId", props(func(p *sinkcfg.RabbitMQProperties) { p.AppID = bad }), true},
		{"userId", props(func(p *sinkcfg.RabbitMQProperties) { p.UserID = bad }), true},
		{"contentType", props(func(p *sinkcfg.RabbitMQProperties) { p.ContentType = bad }), true},
		{"contentEncoding", props(func(p *sinkcfg.RabbitMQProperties) { p.ContentEncoding = bad }), true},
		{"expiration", props(func(p *sinkcfg.RabbitMQProperties) { p.Expiration = bad }), true},
		{"priority", props(func(p *sinkcfg.RabbitMQProperties) { p.Priority = bad }), true},
		{"priority out of range", props(func(p *sinkcfg.RabbitMQProperties) { p.Priority = mustTemplate(t, "256") }), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildMessage(tt.cfg, testContext())
			if err == nil {
				t.Fatal("buildMessage() error = nil, want an error")
			}
			if got := errors.Is(err, template.ErrRender); got != tt.render {
				t.Errorf("buildMessage() error = %v, render = %v, want %v", err, got, tt.render)
			}
		})
	}
}

func TestDeliveryModeByteDefaultsToPersistent(t *testing.T) {
	if got := deliveryModeByte(""); got != amqp.Persistent {
		t.Errorf("deliveryModeByte(\"\") = %d, want persistent", got)
	}
	if got := deliveryModeByte(sinkcfg.RabbitMQDeliveryPersistent); got != amqp.Persistent {
		t.Errorf("deliveryModeByte(persistent) = %d, want persistent", got)
	}
}

func TestWrapErrorClassifiesRabbitErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"exchange missing", errExchangeMissing, true},
		{"exchange mismatch", errExchangeMismatch, true},
		{"unroutable", errUnroutable, true},
		{"nacked", errPublishNacked, true},
		{"access refused", &amqp.Error{Code: amqp.AccessRefused}, true},
		{"not allowed", &amqp.Error{Code: amqp.NotAllowed}, true},
		{"not found", &amqp.Error{Code: amqp.NotFound}, true},
		{"precondition failed", &amqp.Error{Code: amqp.PreconditionFailed}, true},
		{"internal error", &amqp.Error{Code: amqp.InternalError}, true},
		{"connection forced", &amqp.Error{Code: amqp.ConnectionForced}, false},
		{"reconnecting", errReconnecting, false},
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
	if isPermanentRabbitError(nil) {
		t.Error("isPermanentRabbitError(nil) = true, want false")
	}
}

func TestIsTransientConnError(t *testing.T) {
	transient := []error{
		amqp.ErrClosed, context.DeadlineExceeded, context.Canceled, errReconnecting,
		&amqp.Error{Code: amqp.ConnectionForced}, &amqp.Error{Code: amqp.FrameError},
		&amqp.Error{Code: amqp.ChannelError}, &amqp.Error{Code: amqp.UnexpectedFrame},
	}
	for _, err := range transient {
		if !isTransientConnError(err) {
			t.Errorf("isTransientConnError(%v) = false, want true", err)
		}
	}

	for _, err := range []error{nil, errors.New("boom"), &amqp.Error{Code: amqp.NotFound}} {
		if isTransientConnError(err) {
			t.Errorf("isTransientConnError(%v) = true, want false", err)
		}
	}
}

func TestClassifyDeclareErrorKeepsOtherErrors(t *testing.T) {
	for _, err := range []error{errors.New("boom"), &amqp.Error{Code: amqp.ChannelError}} {
		if got := classifyDeclareError(err); got != err { //nolint:errorlint // asserts the very same error value is returned
			t.Errorf("classifyDeclareError(%v) = %v, want it unchanged", err, got)
		}
	}
}

func TestEnsureDeclaredDoesNotCacheTransientFailures(t *testing.T) {
	calls := 0
	r := newExchangeReconciler(func() (*amqp.Channel, error) {
		calls++
		return nil, errReconnecting
	})
	exchange := sinkcfg.RabbitMQExchange{Mode: sinkcfg.RabbitMQExchangeModePassive}

	for range 2 {
		if err := r.ensureDeclared(context.Background(), exchange, "events"); !errors.Is(err, errReconnecting) {
			t.Fatalf("ensureDeclared() error = %v, want errReconnecting", err)
		}
	}
	if calls != 2 {
		t.Errorf("channel requests = %d, want 2", calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.ensureDeclared(ctx, exchange, "events"); !errors.Is(err, context.Canceled) {
		t.Errorf("ensureDeclared(cancelled) error = %v, want context.Canceled", err)
	}
	if calls != 2 {
		t.Errorf("channel requests = %d, want none for a cancelled context", calls)
	}
}

func TestInvalidateResetsState(t *testing.T) {
	r := newExchangeReconciler(nil)
	r.stateFor("events").done = true

	r.invalidate()

	if r.stateFor("events").done {
		t.Error("state still done after invalidate")
	}
}

func TestAwaitPublishResultReportsReturnWhicheverArrivesFirst(t *testing.T) {
	// select picks a ready case at random; 64 rounds reach both branches with
	// a miss probability of 2^-63.
	for range 64 {
		rc := newTestConn(1)
		rc.returns <- amqp.Return{ReplyCode: amqp.NoRoute}
		rc.confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}

		if err := awaitPublishResult(context.Background(), rc, nil); !errors.Is(err, errUnroutable) {
			t.Fatalf("awaitPublishResult() error = %v, want errUnroutable", err)
		}
	}
}

func TestDrainPendingDiscardsStaleResults(t *testing.T) {
	rc := newTestConn(2)
	rc.confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}
	rc.returns <- amqp.Return{ReplyCode: amqp.NoRoute}

	drainPending(rc.confirms, rc.returns)

	if len(rc.confirms) != 0 || len(rc.returns) != 0 {
		t.Errorf("pending = %d confirms, %d returns, want none", len(rc.confirms), len(rc.returns))
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
			args := []string{"-test.run=^TestShutdownClosesClients$"}
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
func diagnosticURL(addr string) string {
	return "amqp://diagnostic-user-sentinel:diagnostic-password-sentinel@" + addr + "/diagnostic-vhost-sentinel?token=diagnostic-query-sentinel"
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

// rabbitSink returns a sink publishing "payload" to exchange "events" with
// routing key "<schema>.<table>", reconciled in create-when-missing mode.
func rabbitSink(t *testing.T, url string) sinkcfg.RabbitMQSink {
	t.Helper()

	return sinkcfg.RabbitMQSink{
		RabbitMQCommon: sinkcfg.RabbitMQCommon{URL: mustTemplate(t, url), Payload: mustTemplate(t, "payload")},
		Exchange: sinkcfg.RabbitMQExchange{
			Name: mustTemplate(t, "events"),
			Type: "topic",
			Mode: sinkcfg.RabbitMQExchangeModeCreateWhenMissing,
		},
		RoutingKey: mustTemplate(t, "{{ .Schema }}.{{ .Table }}"),
	}
}

// notFoundWhenPassive returns a declare hook that answers a passive probe
// with 404 and a real declare with code.
func notFoundWhenPassive(code uint16) func(string, bool) uint16 {
	return func(_ string, passive bool) uint16 {
		if passive {
			return amqp.NotFound
		}
		return code
	}
}

// mustClient returns the cached client for url or fails the test.
func mustClient(t *testing.T, url string) *rabbitConn {
	t.Helper()

	rc, err := getClient(rabbitSink(t, url), testContext())
	if err != nil {
		t.Fatalf("getClient() error = %v", err)
	}

	return rc
}

// injectClient caches rc under its url for the duration of the test.
func injectClient(t *testing.T, rc *rabbitConn) {
	t.Helper()

	clientsMu.Lock()
	clients[clientKey{url: rc.url}] = rc
	clientsMu.Unlock()

	t.Cleanup(func() {
		clientsMu.Lock()
		delete(clients, clientKey{url: rc.url})
		clientsMu.Unlock()
	})
}

// forgetClients stops, closes and evicts the cached client for url when the
// test ends, so tests do not share connections.
func forgetClients(t *testing.T, url string) {
	t.Helper()

	t.Cleanup(func() {
		clientsMu.Lock()
		rc, ok := clients[clientKey{url: url}]
		delete(clients, clientKey{url: url})
		clientsMu.Unlock()

		if !ok {
			return
		}
		select {
		case <-rc.quit:
		default:
			close(rc.quit)
		}

		rc.mu.Lock()
		defer rc.mu.Unlock()
		if rc.conn != nil {
			_ = rc.conn.Close()
		}
	})
}

// isReconnected reports whether rc is ready again on a live connection.
func isReconnected(rc *rabbitConn) bool {
	if !isClosed(rc.readyChan()) {
		return false
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	return rc.conn != nil && !rc.conn.IsClosed()
}

// isClosed reports whether ch is closed.
func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
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

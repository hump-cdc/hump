package config

import (
	"errors"
	"strings"
	"testing"
	"time"

	"hump/internal/port/sink"
	"hump/internal/types/config"
	"hump/internal/types/errs"
	"hump/pkg/retry"
)

func TestParse_KafkaSink(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - kafka:
          url: localhost:9092
          topic: users
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 1 {
		t.Fatalf("want 1 table, got %d", len(cfg.Tables))
	}
	tbl := cfg.Tables[0]
	if tbl.Schema != "public" || tbl.Name != "users" {
		t.Fatalf("want schema=public table=users, got schema=%q table=%q", tbl.Schema, tbl.Name)
	}
	if len(tbl.Sinks) != 1 {
		t.Fatalf("want 1 sink, got %d", len(tbl.Sinks))
	}

	sc := tbl.Sinks[0]
	if sc.Sink == nil {
		t.Fatalf("Sink is nil")
	}
	if sc.Sink.Type() != sink.SinkTypeKafka {
		t.Fatalf("want sink type kafka, got %q", sc.Sink.Type())
	}
	k, ok := sc.Sink.(*sink.KafkaSink)
	if !ok {
		t.Fatalf("want *KafkaSink, got %T", sc.Sink)
	}
	if k.URL.Source() != "localhost:9092" {
		t.Fatalf("want url=localhost:9092, got %q", k.URL.Source())
	}
	if k.Topic.Source() != "users" {
		t.Fatalf("want topic=users, got %q", k.Topic.Source())
	}
}

func TestParse_HTTPSink(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: http://localhost:8080/users
          method: POST
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 1 || len(cfg.Tables[0].Sinks) != 1 {
		t.Fatalf("unexpected shape: %+v", cfg)
	}

	sc := cfg.Tables[0].Sinks[0]
	if sc.Sink.Type() != sink.SinkTypeHTTP {
		t.Fatalf("want sink type http, got %q", sc.Sink.Type())
	}
	h, ok := sc.Sink.(*sink.HTTPSink)
	if !ok {
		t.Fatalf("want *HTTPSink, got %T", sc.Sink)
	}
	if h.URL.Source() != "http://localhost:8080/users" {
		t.Fatalf("want url=http://localhost:8080/users, got %q", h.URL.Source())
	}
	if h.Method.Source() != "POST" {
		t.Fatalf("want method=POST, got %q", h.Method.Source())
	}
}

func TestParse_DefaultsApplied(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: http://x
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	sc := cfg.Tables[0].Sinks[0]

	if sc.Failure == nil {
		t.Fatalf("Failure not initialized to default")
	}
	if sc.Failure.Mode != retry.FailureModeCrash {
		t.Fatalf("want failure mode=crash, got %q", sc.Failure.Mode)
	}

	if sc.Retry == nil {
		t.Fatalf("Retry not initialized to default")
	}
	if sc.Retry.MaxAttempts != 5 {
		t.Fatalf("want retry MaxAttempts=5, got %d", sc.Retry.MaxAttempts)
	}
	if sc.Retry.AttemptTimeout != 10*time.Second {
		t.Fatalf("want retry AttemptTimeout=10s, got %s", sc.Retry.AttemptTimeout)
	}
	if sc.Retry.Backoff.Type != retry.BackoffTypeExponential {
		t.Fatalf("want backoff type=exponential, got %q", sc.Retry.Backoff.Type)
	}
	if sc.Retry.Backoff.Initial != 1*time.Second {
		t.Fatalf("want backoff initial=1s, got %s", sc.Retry.Backoff.Initial)
	}
	if sc.Retry.Backoff.Max != 30*time.Second {
		t.Fatalf("want backoff max=30s, got %s", sc.Retry.Backoff.Max)
	}
	if sc.Retry.Backoff.Multiplier != 2 {
		t.Fatalf("want backoff multiplier=2, got %v", sc.Retry.Backoff.Multiplier)
	}
	if sc.Retry.Backoff.Jitter != retry.JitterModeFull {
		t.Fatalf("want backoff jitter=full, got %q", sc.Retry.Backoff.Jitter)
	}

	if sc.Delivery == nil {
		t.Fatalf("Delivery not initialized to default")
	}
	if sc.Delivery.Mode != retry.DeliveryModeReliable {
		t.Fatalf("want delivery mode=reliable, got %q", sc.Delivery.Mode)
	}
	if sc.Delivery.MaxConcurrent != 64 {
		t.Fatalf("want delivery maxConcurrent=64, got %d", sc.Delivery.MaxConcurrent)
	}
	if sc.Delivery.ConcurrencyKey == nil {
		t.Fatalf("delivery ConcurrencyKey not initialized")
	}
}

func TestParse_ExplicitFailureRetryDelivery(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: http://x
        failure: skip
        retry:
          maxAttempts: 3
          attemptTimeout: 2s
          backoff:
            type: linear
            initial: 500ms
            max: 5s
            increment: 250ms
        delivery:
          mode: async
          maxConcurrent: 8
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	sc := cfg.Tables[0].Sinks[0]
	if sc.Failure.Mode != retry.FailureModeSkip {
		t.Fatalf("want failure mode=skip, got %q", sc.Failure.Mode)
	}
	if sc.Retry.MaxAttempts != 3 {
		t.Fatalf("want retry MaxAttempts=3, got %d", sc.Retry.MaxAttempts)
	}
	if sc.Retry.AttemptTimeout != 2*time.Second {
		t.Fatalf("want retry AttemptTimeout=2s, got %s", sc.Retry.AttemptTimeout)
	}
	if sc.Retry.Backoff.Type != retry.BackoffTypeLinear {
		t.Fatalf("want backoff type=linear, got %q", sc.Retry.Backoff.Type)
	}
	if sc.Retry.Backoff.Increment != 250*time.Millisecond {
		t.Fatalf("want backoff increment=250ms, got %s", sc.Retry.Backoff.Increment)
	}
	if sc.Delivery.Mode != retry.DeliveryModeAsync {
		t.Fatalf("want delivery mode=async, got %q", sc.Delivery.Mode)
	}
	if sc.Delivery.MaxConcurrent != 8 {
		t.Fatalf("want delivery maxConcurrent=8, got %d", sc.Delivery.MaxConcurrent)
	}
}

func TestParse_PresetIsExpanded(t *testing.T) {
	src := `sinks:
  - name: k1
    kafka:
      url: localhost:9092
      topic: orig
tables:
  - schema: public
    table: tags
    sinks:
      - preset: k1
        failure:
          mode: skip
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 1 || len(cfg.Tables[0].Sinks) != 1 {
		t.Fatalf("unexpected shape: %+v", cfg)
	}
	sc := cfg.Tables[0].Sinks[0]
	k, ok := sc.Sink.(*sink.KafkaSink)
	if !ok {
		t.Fatalf("want kafka sink, got %T", sc.Sink)
	}
	if k.URL.Source() != "localhost:9092" {
		t.Fatalf("preset base not applied, url=%q", k.URL.Source())
	}
	if k.Topic.Source() != "orig" {
		t.Fatalf("want topic from preset=orig, got %q", k.Topic.Source())
	}
	if sc.Failure.Mode != retry.FailureModeSkip {
		t.Fatalf("sibling failure override should be applied, got %q", sc.Failure.Mode)
	}
}

func TestParse_NestedTablesAreFlattened(t *testing.T) {
	src := `tables:
  - sinks:
      - http:
          url: http://x
    tables:
      - schema: public
        table: users
      - schema: public
        table: orders
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 2 {
		t.Fatalf("want 2 flattened tables, got %d", len(cfg.Tables))
	}
	names := []string{cfg.Tables[0].Name, cfg.Tables[1].Name}
	want := []string{"users", "orders"}
	for i, w := range want {
		if names[i] != w {
			t.Fatalf("flattened table[%d]: want %q, got %q", i, w, names[i])
		}
		if len(cfg.Tables[i].Sinks) != 1 {
			t.Fatalf("flattened table[%d]: shared sink not propagated", i)
		}
	}
}

func TestParse_ReplicaDefaultsToFull(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: http://x
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Tables[0].Replica != config.ReplicaIdentityFull {
		t.Fatalf("want replica=full by default, got %q", cfg.Tables[0].Replica)
	}
}

func TestParse_ReplicaUnmanaged(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    replica: unmanaged
    sinks:
      - http:
          url: http://x
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Tables[0].Replica != config.ReplicaIdentityUnmanaged {
		t.Fatalf("want replica=unmanaged, got %q", cfg.Tables[0].Replica)
	}
}

func TestParse_ReplicaInvalidValueRejected(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    replica: bogus
    sinks:
      - http:
          url: http://x
`
	if _, err := Parse(src); err == nil {
		t.Fatalf("expected validation error for invalid replica value")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	if _, err := Parse("a: [unterminated\n"); err == nil {
		t.Fatalf("expected error on invalid YAML")
	}
}

func TestParse_UnknownPresetSurfaces(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - preset: missing
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for unknown preset")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected error to mention preset name, got %q", err)
	}
}

func TestParse_NoTablesFailsValidation(t *testing.T) {
	src := `tables: []
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected validation error for empty tables")
	}
}

func TestParse_NoSinksFailsValidation(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks: []
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected validation error for empty sinks")
	}
}

func TestParse_MissingTableNameFailsValidation(t *testing.T) {
	src := `tables:
  - schema: s
    sinks:
      - http:
          url: http://x
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected validation error for missing table name")
	}
}

func TestParse_MissingRequiredSinkFieldFails(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http: {}
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error: http sink requires url")
	}
}

func TestParse_NoSinkDiscriminatorFails(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - url: http://x
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for missing sink discriminator")
	}
}

func TestParse_StrictRejectsUnknownField(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    bogus: 1
    sinks:
      - http:
          url: http://x
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected strict-mode error for unknown field")
	}
}

func TestParse_DuplicateTablesAreMerged(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: http://x
  - schema: public
    table: users
    sinks:
      - http:
          url: http://x
      - kafka:
          url: localhost:9092
          topic: users
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 1 {
		t.Fatalf("want 1 merged table, got %d", len(cfg.Tables))
	}
	if len(cfg.Tables[0].Sinks) != 2 {
		t.Fatalf("want 2 deduplicated sinks, got %d", len(cfg.Tables[0].Sinks))
	}
}

func TestParse_DuplicateSinksWithinTableDeduplicated(t *testing.T) {
	// First table lists the same sink twice; a second entry with the same
	// schema/table forces a merge so mergeTableEntries deduplicates against
	// the first entry's own repeated sinks.
	src := `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: http://x
      - http:
          url: http://x
  - schema: public
    table: users
    sinks:
      - kafka:
          url: localhost:9092
          topic: users
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 1 {
		t.Fatalf("want 1 merged table, got %d", len(cfg.Tables))
	}
	if len(cfg.Tables[0].Sinks) != 2 {
		t.Fatalf("want 2 deduplicated sinks (http, kafka), got %d", len(cfg.Tables[0].Sinks))
	}
}

func TestParse_CrossValidationFails(t *testing.T) {
	// Two tables target the same JetStream (url, stream) with divergent
	// retention, which passes schema validation but fails CrossValidate.
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: FOO
          retention: limits
  - schema: public
    table: orders
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: FOO
          retention: interest
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected cross-validation error")
	}
	if !errors.Is(err, errs.ErrCrossValidate) {
		t.Fatalf("want ErrCrossValidate, got %v", err)
	}
}

func TestParse_EmptySchemaAllowed(t *testing.T) {
	src := `tables:
  - table: t
    sinks:
      - http:
          url: http://x
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Tables[0].Schema != "" {
		t.Fatalf("want empty schema, got %q", cfg.Tables[0].Schema)
	}
	if cfg.Tables[0].Name != "t" {
		t.Fatalf("want table=t, got %q", cfg.Tables[0].Name)
	}
}

func TestParse_NATSCore_Defaults(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	sc := cfg.Tables[0].Sinks[0]
	if sc.Sink.Type() != sink.SinkTypeNatsCore {
		t.Fatalf("want sink type nats-core, got %q", sc.Sink.Type())
	}
	n, ok := sc.Sink.(*sink.NatsCoreSink)
	if !ok {
		t.Fatalf("want *NatsCoreSink, got %T", sc.Sink)
	}
	if n.Subject == nil || n.Subject.Source() != "{{ .Schema }}.{{ .Table }}" {
		var s string
		if n.Subject != nil {
			s = n.Subject.Source()
		}
		t.Fatalf("want default subject template, got %q", s)
	}
	if _, ok := n.Headers["X-Idempotency-Key"]; !ok {
		t.Fatalf("want default header X-Idempotency-Key, got %#v", n.Headers)
	}
	if _, ok := n.Headers["traceparent"]; !ok {
		t.Fatalf("want default header traceparent, got %#v", n.Headers)
	}
}

func TestParse_NATSCore_StreamKeyRejected(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          stream:
            name: X
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for stream key under nats-core")
	}
}

func TestParse_NATSJetStream_Basic(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: FOO
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	sc := cfg.Tables[0].Sinks[0]
	if sc.Sink.Type() != sink.SinkTypeNatsJetStream {
		t.Fatalf("want sink type nats-jetstream, got %q", sc.Sink.Type())
	}
	n, ok := sc.Sink.(*sink.NatsJetStreamSink)
	if !ok {
		t.Fatalf("want *NatsJetStreamSink, got %T", sc.Sink)
	}
	if n.Stream.Name != "FOO" {
		t.Fatalf("want Stream.Name=FOO, got %q", n.Stream.Name)
	}
	if n.Mode != sink.JetStreamReconcileCreateOrUpdate {
		t.Fatalf("want Mode=create-or-update (default), got %q", n.Mode)
	}
}

func TestParse_NATSJetStream_MissingStreamRejected(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for jetstream without stream")
	}
	if !strings.Contains(err.Error(), "stream") {
		t.Fatalf("expected error to mention stream, got %q", err)
	}
}

func TestParse_NATSJetStream_RetentionWithoutSubjectsAccepted(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: X
          retention: limits
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsJetStreamSink)
	if !ok {
		t.Fatalf("want *NatsJetStreamSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.Retention != "limits" {
		t.Fatalf("want retention=limits, got %q", n.Retention)
	}
	if len(n.Stream.Subjects) != 0 {
		t.Fatalf("want no subjects, got %#v", n.Stream.Subjects)
	}
}

func TestParse_NATSJetStream_SubjectsUnderStreamAccepted(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: X
            subjects:
              - x.*
          retention: limits
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsJetStreamSink)
	if !ok {
		t.Fatalf("want *NatsJetStreamSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if len(n.Stream.Subjects) != 1 || n.Stream.Subjects[0] != "x.*" {
		t.Fatalf("want subjects=[x.*], got %+v", n.Stream.Subjects)
	}
}

func TestParse_NATSCore_AuthCredentialsAccepted(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          credentials: /etc/nats/user.creds
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsCoreSink)
	if !ok {
		t.Fatalf("want *NatsCoreSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.Credentials == nil || n.Credentials.Source() != "/etc/nats/user.creds" {
		t.Fatalf("want credentials=/etc/nats/user.creds, got %+v", n.Credentials)
	}
}

func TestParse_NATSCore_AuthNKeySeedAccepted(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          nkeySeed: SUAEXAMPLE
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsCoreSink)
	if !ok {
		t.Fatalf("want *NatsCoreSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.NKeySeed == nil || n.NKeySeed.Source() != "SUAEXAMPLE" {
		t.Fatalf("want nkeySeed=SUAEXAMPLE, got %+v", n.NKeySeed)
	}
}

func TestParse_NATSCore_AuthUserPassAccepted(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          username: u
          password: p
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsCoreSink)
	if !ok {
		t.Fatalf("want *NatsCoreSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.Username == nil || n.Username.Source() != "u" {
		t.Fatalf("want username=u, got %+v", n.Username)
	}
	if n.Password == nil || n.Password.Source() != "p" {
		t.Fatalf("want password=p, got %+v", n.Password)
	}
}

func TestParse_NATSCore_AuthTokenAccepted(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          token: tok
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsCoreSink)
	if !ok {
		t.Fatalf("want *NatsCoreSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.Token == nil || n.Token.Source() != "tok" {
		t.Fatalf("want token=tok, got %+v", n.Token)
	}
}

func TestParse_NATSCore_AuthCredentialsAndTokenRejected(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          credentials: /etc/nats/user.creds
          token: tok
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for multiple auth methods")
	}
	if !strings.Contains(err.Error(), "auth") {
		t.Fatalf("expected error to mention auth, got %q", err)
	}
}

func TestParse_NATSCore_AuthNKeyAndUserPassRejected(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          nkeySeed: SUAEXAMPLE
          username: u
          password: p
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for multiple auth methods")
	}
	if !strings.Contains(err.Error(), "auth") {
		t.Fatalf("expected error to mention auth, got %q", err)
	}
}

func TestParse_NATSCore_AuthUserPassAndTokenRejected(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-core:
          url: nats://localhost:4222
          username: u
          password: p
          token: tok
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for multiple auth methods")
	}
	if !strings.Contains(err.Error(), "auth") {
		t.Fatalf("expected error to mention auth, got %q", err)
	}
}

func TestParse_NATSJetStream_ReconcileModeCreateWhenMissing(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: X
          mode: create-when-missing
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sink.NatsJetStreamSink)
	if !ok {
		t.Fatalf("want *NatsJetStreamSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.Mode != sink.JetStreamReconcileCreateWhenMissing {
		t.Fatalf("want Mode=create-when-missing, got %q", n.Mode)
	}
}

func TestParse_NATSJetStream_ReconcileModeInvalid(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: X
          mode: foo
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for invalid jetstream mode")
	}
	if !strings.Contains(err.Error(), "mode") {
		t.Fatalf("expected error to mention mode, got %q", err)
	}
}

func TestParse_NATSJetStream_RetentionEnumInvalid(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: X
            subjects:
              - x.*
          retention: bogus
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for invalid retention")
	}
	if !strings.Contains(err.Error(), "retention") {
		t.Fatalf("expected error to mention retention, got %q", err)
	}
}

func TestParse_NATSJetStream_StorageEnumInvalid(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: X
            subjects:
              - x.*
          storage: bogus
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for invalid storage")
	}
	if !strings.Contains(err.Error(), "storage") {
		t.Fatalf("expected error to mention storage, got %q", err)
	}
}

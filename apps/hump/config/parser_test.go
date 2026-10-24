package config

import (
	errs "errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"hump/config/errors"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/sink"
	"hump/pkg/resilience"

	"github.com/goccy/go-yaml"
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
	k, ok := sc.Sink.(*sinkcfg.KafkaSink)
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
	h, ok := sc.Sink.(*sinkcfg.HTTPSink)
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
	if sc.Failure.Mode != resilience.FailureModeCrash {
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
	if sc.Retry.Backoff.Type != resilience.BackoffTypeExponential {
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
	if sc.Retry.Backoff.Jitter != resilience.JitterModeFull {
		t.Fatalf("want backoff jitter=full, got %q", sc.Retry.Backoff.Jitter)
	}
}

func TestParse_ExplicitFailureRetry(t *testing.T) {
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
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	sc := cfg.Tables[0].Sinks[0]
	if sc.Failure.Mode != resilience.FailureModeSkip {
		t.Fatalf("want failure mode=skip, got %q", sc.Failure.Mode)
	}
	if sc.Retry.MaxAttempts != 3 {
		t.Fatalf("want retry MaxAttempts=3, got %d", sc.Retry.MaxAttempts)
	}
	if sc.Retry.AttemptTimeout != 2*time.Second {
		t.Fatalf("want retry AttemptTimeout=2s, got %s", sc.Retry.AttemptTimeout)
	}
	if sc.Retry.Backoff.Type != resilience.BackoffTypeLinear {
		t.Fatalf("want backoff type=linear, got %q", sc.Retry.Backoff.Type)
	}
	if sc.Retry.Backoff.Increment != 250*time.Millisecond {
		t.Fatalf("want backoff increment=250ms, got %s", sc.Retry.Backoff.Increment)
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
	k, ok := sc.Sink.(*sinkcfg.KafkaSink)
	if !ok {
		t.Fatalf("want kafka sink, got %T", sc.Sink)
	}
	if k.URL.Source() != "localhost:9092" {
		t.Fatalf("preset base not applied, url=%q", k.URL.Source())
	}
	if k.Topic.Source() != "orig" {
		t.Fatalf("want topic from preset=orig, got %q", k.Topic.Source())
	}
	if sc.Failure.Mode != resilience.FailureModeSkip {
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
	names := []domain.TableName{cfg.Tables[0].Name, cfg.Tables[1].Name}
	want := []domain.TableName{"users", "orders"}
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
	if cfg.Tables[0].Replica != domain.ReplicaIdentityFull {
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
	if cfg.Tables[0].Replica != domain.ReplicaIdentityUnmanaged {
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
	if !errs.Is(err, errors.ErrCrossValidate) {
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
	n, ok := sc.Sink.(*sinkcfg.NatsCoreSink)
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
	n, ok := sc.Sink.(*sinkcfg.NatsJetStreamSink)
	if !ok {
		t.Fatalf("want *NatsJetStreamSink, got %T", sc.Sink)
	}
	if n.Stream.Name != "FOO" {
		t.Fatalf("want Stream.Name=FOO, got %q", n.Stream.Name)
	}
	if n.Mode != sinkcfg.JetStreamReconcileCreateOrUpdate {
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsJetStreamSink)
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsJetStreamSink)
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsCoreSink)
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsCoreSink)
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsCoreSink)
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsCoreSink)
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
	n, ok := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsJetStreamSink)
	if !ok {
		t.Fatalf("want *NatsJetStreamSink, got %T", cfg.Tables[0].Sinks[0].Sink)
	}
	if n.Mode != sinkcfg.JetStreamReconcileCreateWhenMissing {
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

func TestConfiguration_ResolveAll(t *testing.T) {
	cfg := Configuration{Tables: []Table{{Name: "users"}, {Name: "orders"}}}

	got := cfg.ResolveAll()
	if len(got) != 2 || got[0].Name != "users" || got[1].Name != "orders" {
		t.Fatalf("ResolveAll = %+v, want the configured tables in order", got)
	}
}

func TestParse_EmptySinkEntryRejected(t *testing.T) {
	for _, sinks := range []string{
		"    sinks: [~]\n",
		"    sinks:\n      -\n",
		"    sinks:\n      - http:\n          url: http://x\n      - null\n",
	} {
		_, err := Parse("tables:\n  - schema: public\n    table: users\n" + sinks)
		if err == nil {
			t.Fatalf("Parse with %q: want an error for the empty sink entry", sinks)
		}

		if !strings.Contains(err.Error(), "of table public.users is empty") {
			t.Fatalf("Parse with %q error = %q, want it to name the empty entry", sinks, err)
		}
	}
}

func TestParse_BlockScalarTemplateKeepsItsValue(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{body: "|\n            line one\n            line two\n", want: "line one\nline two\n"},
		{body: "|-\n            id={{ .Next.id }}\n", want: "id={{ .Next.id }}"},
		{body: ">-\n            a\n            b\n", want: "a b"},
	}

	for _, tt := range tests {
		cfg, err := Parse("tables:\n  - schema: public\n    table: users\n    sinks:\n      - http:\n" +
			"          url: http://x\n          body: " + tt.body + "          headers:\n            X-A: " +
			strings.ReplaceAll(tt.body, "\n            ", "\n              "))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}

		h := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.HTTPSink)
		if got := h.Body.Source(); got != tt.want {
			t.Errorf("body %q = %q, want %q", tt.body, got, tt.want)
		}
		if got := h.Headers["X-A"].Source(); got != tt.want {
			t.Errorf("header %q = %q, want %q", tt.body, got, tt.want)
		}
	}
}

func TestParse_WhitespaceTemplateSurvivesExpand(t *testing.T) {
	for _, value := range []string{`"\t"`, `"\t\t"`, `"a\t"`, `"\n"`, `" \t"`} {
		cfg, err := Parse("tables:\n  - schema: public\n    table: users\n    sinks:\n      - http:\n" +
			"          url: http://x\n          body: " + value + "\n          headers:\n            X-A: " + value + "\n")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}

		want, err := strconv.Unquote(value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", value, err)
		}

		h := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.HTTPSink)
		if got := h.Body.Source(); got != want {
			t.Errorf("body %s = %q, want %q", value, got, want)
		}
		if header := h.Headers["X-A"]; header == nil || header.Source() != want {
			t.Errorf("header %s = %v, want %q", value, header, want)
		}
	}
}

func TestParse_FirstUnknownFieldInDocumentOrder(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "top level", src: "zzz: 1\naaa: 2\n", want: `unknown field "zzz"`},
		{
			name: "table before its sinks",
			src:  "tables:\n  - schema: public\n    table: users\n    zzz: 1\n    aaa: 2\n    sinks: [{http: {url: http://x, q: 1}}]\n",
			want: `unknown field "zzz"`,
		},
		{
			name: "within a sink",
			src:  "tables:\n  - schema: public\n    table: users\n    sinks: [{http: {url: http://x, zz: 1, aa: 2, mm: 3}}]\n",
			want: `unknown field "zz"`,
		},
		{
			name: "nested plain struct",
			src:  "tables:\n  - schema: public\n    table: users\n    sinks: [{rabbitmq: {url: amqp://x, exchange: {zz: 1, name: e, aa: 2}}}]\n",
			want: `unknown field "zz"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 50 {
				_, err := Parse(tt.src)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("Parse error = %v, want %s", err, tt.want)
				}
			}
		})
	}
}

func TestParse_StringFieldsKeepScalarText(t *testing.T) {
	cfg, err := Parse(`tables:
  - schema: 007
    table: 00000
    sinks:
      - nats-jetstream:
          url: nats://localhost:4222
          stream:
            name: 0x1F
            subjects: [1.50, true]
  - schema: .inf
    table: 1e3
    sinks:
      - http:
          url: http://x
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := cfg.Tables[0]; got.Schema != "007" || got.Name != "00000" {
		t.Errorf("table = %q.%q, want 007.00000", got.Schema, got.Name)
	}
	if got := cfg.Tables[1]; got.Schema != ".inf" || got.Name != "1e3" {
		t.Errorf("table = %q.%q, want .inf.1e3", got.Schema, got.Name)
	}

	js := cfg.Tables[0].Sinks[0].Sink.(*sinkcfg.NatsJetStreamSink)
	if js.Stream.Name != "0x1F" || !slices.Equal(js.Stream.Subjects, []string{"1.50", "true"}) {
		t.Errorf("stream = %q %q, want 0x1F [1.50 true]", js.Stream.Name, js.Stream.Subjects)
	}
}

func TestParse_NumericTableNamesStayDistinct(t *testing.T) {
	cfg, err := Parse("tables:\n  - table: 1\n    sinks: [{http: {url: http://a}}]\n  - table: 2\n    sinks: [{http: {url: http://b}}]\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(cfg.Tables) != 2 || cfg.Tables[0].Name != "1" || cfg.Tables[1].Name != "2" {
		t.Fatalf("tables = %+v, want 1 and 2 kept apart", cfg.Tables)
	}
}

func FuzzParse(f *testing.F) {
	for _, src := range []string{
		"",
		"# only a comment\n",
		"tables: []\n",
		"tables:\n  - schema: public\n    table: users\n    sinks:\n      - kafka:\n          url: localhost:9092\n          topic: users\n",
		"tables:\n  - schema: public\n    table: users\n    replica: unmanaged\n    operations: [insert, delete]\n    sinks:\n" +
			"      - http:\n          url: https://example.com/{{ .Table }}\n          headers:\n            X-Id: \"{{ .Next.id }}\"\n" +
			"        failure: skip\n        retry:\n          maxAttempts: 3\n          backoff:\n            type: linear\n",
		"sinks:\n  - name: k1\n    kafka:\n      url: localhost:9092\n      topic: orig\ntables:\n  - schema: public\n    table: tags\n" +
			"    sinks:\n      - preset: k1\n        failure:\n          mode: skip\n",
		"tables:\n  - sinks:\n      - http:\n          url: http://x\n    tables:\n      - schema: public\n        table: users\n" +
			"      - schema: public\n        table: orders\n",
		"tables:\n  - schema: public\n    table: a\n    sinks:\n      - nats-jetstream:\n          url: nats://localhost:4222\n" +
			"          stream:\n            name: X\n            subjects: [x.*]\n          retention: limits\n" +
			"  - schema: public\n    table: b\n    sinks:\n      - nats-jetstream:\n          url: nats://localhost:4222\n" +
			"          stream:\n            name: X\n            subjects: [y.*]\n",
		"tables:\n  - schema: public\n    table: users\n    sinks:\n      - rabbitmq:\n          url: amqp://localhost\n" +
			"          exchange:\n            name: hump\n            type: topic\n      - rabbitmq-stream:\n          url: rabbitmq-stream://localhost\n",
		"tables:\n  - &t\n    schema: public\n    table: users\n    sinks: [{nats-core: {url: nats://x, auth: {token: t, user: u}}}]\n  - *t\n",
		"tables: {schema: public}\n",
		"[1, 2, 3]\n",
		"tables:\n  - schema: public\n    table: users\n    sinks: [~]\n",
		"tables:\n  - #0000000000000\n    table: 00000\n    sinks:\n      -",
		"1:\n0: ",
		"tables:\n  - schema: public\n    table: users\n    zzz: 1\n    aaa: 2\n    sinks: [{http: {url: http://x, q: 1}}]\n",
		"tables:\n  - schema: 007\n    table: 00000\n    sinks:\n      - http:\n          url: http://x\n" +
			"          body: \"\\t\"\n          headers:\n            X-A: |\n              line one\n              line two\n",
	} {
		f.Add(src)
	}

	f.Fuzz(func(t *testing.T, src string) {
		cfg, err := Parse(src)

		again, againErr := Parse(src)
		if (err == nil) != (againErr == nil) || err != nil && err.Error() != againErr.Error() {
			t.Fatalf("Parse(%q) error = %v, then %v, want it deterministic", src, err, againErr)
		}

		if err != nil {
			return
		}

		dump, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("yaml.Marshal of Parse(%q) error = %v", src, err)
		}

		if againDump, err := yaml.Marshal(again); err != nil || string(againDump) != string(dump) {
			t.Fatalf("Parse(%q) dumps as %q, then %q, %v, want it deterministic", src, dump, againDump, err)
		}

		// A document without a mapping parses to no tables at all, which the
		// dump then spells out as an empty list that validation rejects.
		if len(cfg.Tables) == 0 {
			return
		}

		for _, table := range cfg.Tables {
			for i, sink := range table.Sinks {
				if sink.Sink == nil || sink.Failure == nil || sink.Retry == nil {
					t.Fatalf("Parse(%q) sink %d of %s.%s = %+v, want its definition and policies set", src, i, table.Schema, table.Name, sink)
				}
			}
		}

		if _, err := Parse(string(dump)); err != nil {
			t.Fatalf("Parse(%q) dumps as %q, which does not parse again: %v", src, dump, err)
		}
	})
}

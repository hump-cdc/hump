package config

import (
	"errors"
	"strings"
	"testing"
	"time"

	"hump/internal/template"
	"hump/internal/types/config"
	"hump/internal/types/errs"
	"hump/internal/types/sink"

	"github.com/goccy/go-yaml"
)

// mustTpl builds a template.Template from src, failing the test on error.
func mustTpl(t *testing.T, src string) *template.Template {
	t.Helper()
	tpl, err := template.New(src)
	if err != nil {
		t.Fatalf("template.New(%q): %v", src, err)
	}
	return tpl
}

// parseSink parses a single-sink config and returns the resulting sink.Sink.
func parseSink(t *testing.T, sinkYAML string) sink.Sink {
	t.Helper()
	src := "tables:\n  - schema: public\n    table: users\n    sinks:\n      - " +
		strings.ReplaceAll(sinkYAML, "\n", "\n        ") + "\n"
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse:\n%s\nerr: %v", src, err)
	}
	if len(cfg.Tables) != 1 || len(cfg.Tables[0].Sinks) != 1 {
		t.Fatalf("unexpected shape: %+v", cfg)
	}
	return cfg.Tables[0].Sinks[0].Sink
}

// --- Type() and DisplayName() ---------------------------------------------

func TestSink_TypeAndDisplayName(t *testing.T) {
	url := func() *template.Template { return mustTpl(t, "endpoint") }

	cases := []struct {
		name        string
		populated   interface{ DisplayName() string }
		nilRecv     interface{ DisplayName() string }
		wantType    sink.SinkType
		wantDisplay string
	}{
		{
			name:        "http",
			populated:   &config.HTTPSink{URL: url()},
			nilRecv:     (*config.HTTPSink)(nil),
			wantType:    sink.SinkTypeHTTP,
			wantDisplay: "http(endpoint)",
		},
		{
			name:        "kafka",
			populated:   &config.KafkaSink{URL: url()},
			nilRecv:     (*config.KafkaSink)(nil),
			wantType:    sink.SinkTypeKafka,
			wantDisplay: "kafka(endpoint)",
		},
		{
			name:        "nats-core",
			populated:   &config.NatsCoreSink{NatsCommon: config.NatsCommon{URL: url()}},
			nilRecv:     (*config.NatsCoreSink)(nil),
			wantType:    sink.SinkTypeNatsCore,
			wantDisplay: "nats-core(endpoint)",
		},
		{
			name:        "nats-jetstream",
			populated:   &config.NatsJetStreamSink{NatsCommon: config.NatsCommon{URL: url()}},
			nilRecv:     (*config.NatsJetStreamSink)(nil),
			wantType:    sink.SinkTypeNatsJetStream,
			wantDisplay: "nats-jetstream(endpoint)",
		},
		{
			name:        "rabbitmq",
			populated:   &config.RabbitMQSink{RabbitMQCommon: config.RabbitMQCommon{URL: url()}},
			nilRecv:     (*config.RabbitMQSink)(nil),
			wantType:    sink.SinkTypeRabbitMQ,
			wantDisplay: "rabbitmq(endpoint)",
		},
		{
			name:        "rabbitmq-stream",
			populated:   &config.RabbitMQStreamSink{RabbitMQCommon: config.RabbitMQCommon{URL: url()}},
			nilRecv:     (*config.RabbitMQStreamSink)(nil),
			wantType:    sink.SinkTypeRabbitMQStream,
			wantDisplay: "rabbitmq-stream(endpoint)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.populated.(sink.Sink)
			if s.Type() != tc.wantType {
				t.Fatalf("Type: want %q, got %q", tc.wantType, s.Type())
			}
			if got := tc.populated.DisplayName(); got != tc.wantDisplay {
				t.Fatalf("DisplayName populated: want %q, got %q", tc.wantDisplay, got)
			}
			// Nil receiver and empty URL fall back to the bare type name.
			if got := tc.nilRecv.DisplayName(); got != string(tc.wantType) {
				t.Fatalf("DisplayName nil: want %q, got %q", tc.wantType, got)
			}
		})
	}
}

// --- SinkConfig.MarshalYAML ------------------------------------------------

func TestSinkConfig_MarshalYAML_RoundTrip(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - kafka:
          url: localhost:9092
          topic: users
        failure: skip
        retry:
          maxAttempts: 3
        delivery:
          mode: async
`
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Marshalling exercises SinkConfig.MarshalYAML for every non-nil section.
	out, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, want := range []string{"kafka", "failure", "retry", "delivery"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("marshalled output missing %q:\n%s", want, out)
		}
	}
}

func TestSinkConfig_MarshalYAML_EmptyOmitsAll(t *testing.T) {
	// A zero SinkConfig has no sink/failure/retry/delivery; MarshalYAML must
	// return an empty mapping without panicking.
	out, err := yaml.Marshal(config.SinkConfig{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(out), "failure") {
		t.Fatalf("empty SinkConfig should not emit failure: %s", out)
	}
}

// --- Failure UnmarshalYAML branches ----------------------------------------

func TestParse_Failure_ScalarNonString(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        failure: 123
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for non-string scalar failure")
	}
	if !strings.Contains(err.Error(), "string or mapping") {
		t.Fatalf("expected 'string or mapping' error, got %q", err)
	}
}

func TestParse_Failure_ScalarInvalidMode(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        failure: bogus
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected validation error for invalid failure mode")
	}
}

func TestParse_Failure_MappingUnknownField(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        failure:
          bogus: 1
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected strict error for unknown failure field")
	}
}

// --- Delivery / Retry / Backoff decode-error branches ----------------------

func TestParse_Delivery_UnknownField(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        delivery:
          bogus: 1
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected strict error for unknown delivery field")
	}
}

func TestParse_Retry_UnknownField(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        retry:
          bogus: 1
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected strict error for unknown retry field")
	}
}

func TestParse_Backoff_UnknownField(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        retry:
          backoff:
            bogus: 1
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected strict error for unknown backoff field")
	}
}

// --- SinkConfig.UnmarshalYAML / unmarshalSink / getSinkType branches --------

func TestParse_SinkNotAMapping(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - just-a-scalar
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error: sink must be a mapping")
	}
	if !strings.Contains(err.Error(), "mapping") {
		t.Fatalf("expected mapping error, got %q", err)
	}
}

func TestParse_SinkUnknownSiblingField(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        bogus: 1
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected strict error for unknown sink sibling field")
	}
}

func TestParse_MultipleSinkDiscriminatorsRejected(t *testing.T) {
	src := `tables:
  - schema: s
    table: t
    sinks:
      - http:
          url: http://x
        kafka:
          url: localhost:9092
          topic: t
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for multiple sink discriminators")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("expected 'exactly one' error, got %q", err)
	}
}

// unmarshalSink returns the inner decode error for each sink type when a
// strict-mode unknown field is present inside the discriminator value.
func TestParse_SinkInnerDecodeErrors(t *testing.T) {
	cases := []struct {
		name      string
		discrim   string
		innerYAML string
	}{
		{"kafka", "kafka", "url: localhost:9092\n          bogus: 1"},
		{"nats-core", "nats-core", "url: nats://x\n          bogus: 1"},
		{"nats-jetstream", "nats-jetstream", "url: nats://x\n          stream:\n            name: X\n          bogus: 1"},
		{"http", "http", "url: http://x\n          bogus: 1"},
		{"rabbitmq", "rabbitmq", "url: amqp://x\n          bogus: 1"},
		{"rabbitmq-stream", "rabbitmq-stream", "url: rabbitmq-stream://x\n          bogus: 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "tables:\n  - schema: s\n    table: t\n    sinks:\n      - " +
				tc.discrim + ":\n          " + tc.innerYAML + "\n"
			if _, err := Parse(src); err == nil {
				t.Fatalf("expected inner decode error for %s", tc.name)
			}
		})
	}
}

// --- Kafka sink topic default ----------------------------------------------

func TestParse_Kafka_TopicDefaultApplied(t *testing.T) {
	// Omitting topic exercises the KafkaSink.setDefaults topic-default branch.
	s := parseSink(t, "kafka:\n  url: localhost:9092")
	k, ok := s.(*config.KafkaSink)
	if !ok {
		t.Fatalf("want *KafkaSink, got %T", s)
	}
	if k.Topic == nil || k.Topic.Source() == "" {
		t.Fatalf("kafka topic default not applied")
	}
}

// --- RabbitMQ classic sink: parse, defaults, common defaults ---------------

func TestParse_RabbitMQ_Defaults(t *testing.T) {
	s := parseSink(t, "rabbitmq:\n  url: amqp://localhost")
	r, ok := s.(*config.RabbitMQSink)
	if !ok {
		t.Fatalf("want *RabbitMQSink, got %T", s)
	}
	if r.Type() != sink.SinkTypeRabbitMQ {
		t.Fatalf("want type rabbitmq, got %q", r.Type())
	}
	if r.Exchange.Name == nil || r.Exchange.Name.Source() == "" {
		t.Fatalf("exchange name default not applied")
	}
	if r.Exchange.Type != "topic" {
		t.Fatalf("want exchange type=topic, got %q", r.Exchange.Type)
	}
	if r.Exchange.Durable == nil || !*r.Exchange.Durable {
		t.Fatalf("want exchange durable=true default")
	}
	if r.Exchange.Mode != config.RabbitMQExchangeModeCreateWhenMissing {
		t.Fatalf("want exchange mode=create-when-missing, got %q", r.Exchange.Mode)
	}
	if r.RoutingKey == nil || r.RoutingKey.Source() == "" {
		t.Fatalf("routing key default not applied")
	}
	if r.Mandatory == nil || !*r.Mandatory {
		t.Fatalf("want mandatory=true default")
	}
	if r.DeliveryMode != config.RabbitMQDeliveryPersistent {
		t.Fatalf("want deliveryMode=persistent, got %q", r.DeliveryMode)
	}
	if r.Properties.MessageID == nil || r.Properties.AppID == nil || r.Properties.ContentType == nil {
		t.Fatalf("property defaults not applied: %+v", r.Properties)
	}
	if r.Properties.AppID.Source() != "hump" {
		t.Fatalf("want appId=hump, got %q", r.Properties.AppID.Source())
	}
	// RabbitMQCommon defaults.
	if r.Payload == nil || r.Payload.Source() == "" {
		t.Fatalf("payload default not applied")
	}
	if _, ok := r.Headers["X-Idempotency-Key"]; !ok {
		t.Fatalf("want default header X-Idempotency-Key, got %#v", r.Headers)
	}
	if _, ok := r.Headers["traceparent"]; !ok {
		t.Fatalf("want default header traceparent, got %#v", r.Headers)
	}
}

func TestParse_RabbitMQ_ExplicitOverrides(t *testing.T) {
	s := parseSink(t, `rabbitmq:
  url: amqp://localhost
  exchange:
    name: events
    type: direct
    durable: false
    autoDelete: true
    mode: passive
    arguments:
      x-key: v
  routingKey: rk
  mandatory: false
  deliveryMode: transient
  properties:
    appId: custom`)
	r := s.(*config.RabbitMQSink)
	if r.Exchange.Type != "direct" {
		t.Fatalf("want type=direct, got %q", r.Exchange.Type)
	}
	if r.Exchange.Durable == nil || *r.Exchange.Durable {
		t.Fatalf("want durable=false")
	}
	if !r.Exchange.AutoDelete {
		t.Fatalf("want autoDelete=true")
	}
	if r.Exchange.Mode != config.RabbitMQExchangeModePassive {
		t.Fatalf("want mode=passive, got %q", r.Exchange.Mode)
	}
	if r.DeliveryMode != config.RabbitMQDeliveryTransient {
		t.Fatalf("want deliveryMode=transient, got %q", r.DeliveryMode)
	}
	if r.Mandatory == nil || *r.Mandatory {
		t.Fatalf("want mandatory=false")
	}
	if r.Properties.AppID.Source() != "custom" {
		t.Fatalf("want appId=custom, got %q", r.Properties.AppID.Source())
	}
}

func TestParse_RabbitMQ_InvalidExchangeType(t *testing.T) {
	_, err := Parse(`tables:
  - schema: s
    table: t
    sinks:
      - rabbitmq:
          url: amqp://x
          exchange:
            type: bogus
`)
	if err == nil {
		t.Fatalf("expected error for invalid exchange type")
	}
}

// --- RabbitMQ Streams sink: parse, defaults --------------------------------

func TestParse_RabbitMQStream_Defaults(t *testing.T) {
	s := parseSink(t, "rabbitmq-stream:\n  url: rabbitmq-stream://localhost")
	r, ok := s.(*config.RabbitMQStreamSink)
	if !ok {
		t.Fatalf("want *RabbitMQStreamSink, got %T", s)
	}
	if r.Type() != sink.SinkTypeRabbitMQStream {
		t.Fatalf("want type rabbitmq-stream, got %q", r.Type())
	}
	if r.Stream.Name == nil || r.Stream.Name.Source() == "" {
		t.Fatalf("stream name default not applied")
	}
	if r.Stream.Mode != config.RabbitMQStreamModeCreateWhenMissing {
		t.Fatalf("want stream mode=create-when-missing, got %q", r.Stream.Mode)
	}
	if r.Stream.MaxAge != 24*time.Hour {
		t.Fatalf("want maxAge=24h, got %s", r.Stream.MaxAge)
	}
	if r.Stream.MaxLengthBytes != 2*1024*1024*1024 {
		t.Fatalf("want maxLengthBytes=2GiB, got %d", r.Stream.MaxLengthBytes)
	}
	if r.Stream.MaxSegmentSizeBytes != 500*1024*1024 {
		t.Fatalf("want maxSegmentSizeBytes=500MiB, got %d", r.Stream.MaxSegmentSizeBytes)
	}
	if r.Payload == nil || r.Payload.Source() == "" {
		t.Fatalf("payload default not applied")
	}
}

func TestParse_RabbitMQStream_ExplicitOverrides(t *testing.T) {
	s := parseSink(t, `rabbitmq-stream:
  url: rabbitmq-stream://localhost
  stream:
    name: my-stream
    mode: passive
    maxAge: 1h
    maxLengthBytes: 1024
    maxSegmentSizeBytes: 512`)
	r := s.(*config.RabbitMQStreamSink)
	if r.Stream.Mode != config.RabbitMQStreamModePassive {
		t.Fatalf("want mode=passive, got %q", r.Stream.Mode)
	}
	if r.Stream.MaxAge != time.Hour {
		t.Fatalf("want maxAge=1h, got %s", r.Stream.MaxAge)
	}
	if r.Stream.MaxLengthBytes != 1024 {
		t.Fatalf("want maxLengthBytes=1024, got %d", r.Stream.MaxLengthBytes)
	}
	if r.Stream.MaxSegmentSizeBytes != 512 {
		t.Fatalf("want maxSegmentSizeBytes=512, got %d", r.Stream.MaxSegmentSizeBytes)
	}
}

func TestParse_RabbitMQStream_InvalidMode(t *testing.T) {
	_, err := Parse(`tables:
  - schema: s
    table: t
    sinks:
      - rabbitmq-stream:
          url: rabbitmq-stream://x
          stream:
            mode: bogus
`)
	if err == nil {
		t.Fatalf("expected error for invalid stream mode")
	}
}

// --- RabbitMQ classic CrossValidate ----------------------------------------

func TestParse_RabbitMQ_CrossValidate_Agreement(t *testing.T) {
	// Two tables target the same (URL, exchange) with identical arguments;
	// CrossValidate must pass.
	src := `tables:
  - schema: public
    table: users
    sinks:
      - rabbitmq:
          url: amqp://localhost
          exchange:
            name: ex
            arguments:
              x-a: 1
              x-b: 2
  - schema: public
    table: orders
    sinks:
      - rabbitmq:
          url: amqp://localhost
          exchange:
            name: ex
            arguments:
              x-b: 2
              x-a: 1
`
	if _, err := Parse(src); err != nil {
		t.Fatalf("expected agreement, got error: %v", err)
	}
}

func TestParse_RabbitMQ_CrossValidate_Divergent(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - rabbitmq:
          url: amqp://localhost
          exchange:
            name: ex
            type: direct
  - schema: public
    table: orders
    sinks:
      - rabbitmq:
          url: amqp://localhost
          exchange:
            name: ex
            type: topic
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected cross-validation error")
	}
	if !errors.Is(err, errs.ErrCrossValidate) {
		t.Fatalf("want ErrCrossValidate, got %v", err)
	}
}

func TestParse_RabbitMQ_CrossValidate_UnmarshalableArguments(t *testing.T) {
	// A non-finite float in exchange.arguments survives parsing but fails
	// json.Marshal inside CrossValidate, exercising the encode-error branch.
	src := `tables:
  - schema: public
    table: users
    sinks:
      - rabbitmq:
          url: amqp://localhost
          exchange:
            name: ex
            arguments:
              x: .nan
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected error for unmarshalable exchange arguments")
	}
	if !errors.Is(err, errs.ErrCrossValidate) {
		t.Fatalf("want ErrCrossValidate, got %v", err)
	}
}

// --- RabbitMQ Streams CrossValidate ----------------------------------------

func TestParse_RabbitMQStream_CrossValidate_Agreement(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - rabbitmq-stream:
          url: rabbitmq-stream://localhost
          stream:
            name: st
            maxAge: 1h
  - schema: public
    table: orders
    sinks:
      - rabbitmq-stream:
          url: rabbitmq-stream://localhost
          stream:
            name: st
            maxAge: 1h
`
	if _, err := Parse(src); err != nil {
		t.Fatalf("expected agreement, got error: %v", err)
	}
}

func TestParse_RabbitMQStream_CrossValidate_Divergent(t *testing.T) {
	src := `tables:
  - schema: public
    table: users
    sinks:
      - rabbitmq-stream:
          url: rabbitmq-stream://localhost
          stream:
            name: st
            maxAge: 1h
  - schema: public
    table: orders
    sinks:
      - rabbitmq-stream:
          url: rabbitmq-stream://localhost
          stream:
            name: st
            maxAge: 2h
`
	_, err := Parse(src)
	if err == nil {
		t.Fatalf("expected cross-validation error")
	}
	if !errors.Is(err, errs.ErrCrossValidate) {
		t.Fatalf("want ErrCrossValidate, got %v", err)
	}
}

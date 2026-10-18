package sink

import (
	errs "errors"
	"slices"
	"strings"
	"testing"

	"hump/config/errors"
	"hump/config/resilience"
	"hump/internal/domain"
	sinkport "hump/internal/port/sink"
	pkgresilience "hump/pkg/resilience"

	"github.com/goccy/go-yaml"
)

func TestConfigUnmarshalEachSinkType(t *testing.T) {
	tests := []struct {
		src  string
		want sinkport.Type
	}{
		{src: "kafka:\n  url: localhost:9092\n", want: sinkport.SinkTypeKafka},
		{src: "nats-core:\n  url: nats://localhost:4222\n", want: sinkport.SinkTypeNatsCore},
		{src: "nats-jetstream:\n  url: nats://localhost:4222\n  stream:\n    name: S\n", want: sinkport.SinkTypeNatsJetStream},
		{src: "http:\n  url: https://example.com\n", want: sinkport.SinkTypeHTTP},
		{src: "rabbitmq:\n  url: amqp://localhost\n", want: sinkport.SinkTypeRabbitMQ},
		{src: "rabbitmq-stream:\n  url: rabbitmq-stream://localhost\n", want: sinkport.SinkTypeRabbitMQStream},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			cfg := unmarshalConfig(t, tt.src)
			if cfg.Sink == nil || cfg.Sink.Type() != tt.want {
				t.Fatalf("Sink = %#v, want type %q", cfg.Sink, tt.want)
			}
		})
	}
}

func TestConfigUnmarshalDefaults(t *testing.T) {
	cfg := unmarshalConfig(t, "http:\n  url: https://example.com\n")

	if cfg.Failure == nil || cfg.Failure.Mode != pkgresilience.FailureModeCrash {
		t.Errorf("Failure = %+v, want crash", cfg.Failure)
	}
	if cfg.Retry == nil || cfg.Retry.MaxAttempts != 5 {
		t.Errorf("Retry = %+v, want defaults", cfg.Retry)
	}
	if !slices.Equal(cfg.Operations, domain.OperationTypes) {
		t.Errorf("Operations = %v, want %v", cfg.Operations, domain.OperationTypes)
	}
}

func TestConfigUnmarshalExplicitValues(t *testing.T) {
	src := "http:\n  url: https://example.com\nfailure: skip\nretry:\n  maxAttempts: 2\noperations: [insert]\n"
	cfg := unmarshalConfig(t, src)

	if cfg.Failure.Mode != pkgresilience.FailureModeSkip {
		t.Errorf("Failure.Mode = %q, want skip", cfg.Failure.Mode)
	}
	if cfg.Retry.MaxAttempts != 2 {
		t.Errorf("Retry.MaxAttempts = %d, want 2", cfg.Retry.MaxAttempts)
	}
	if !slices.Equal(cfg.Operations, []domain.OperationType{domain.OperationTypeInsert}) {
		t.Errorf("Operations = %v, want [insert]", cfg.Operations)
	}
}

func TestConfigUnmarshalRejected(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantMsg string
	}{
		{name: "not a mapping", src: "hello", wantMsg: "must be a mapping"},
		{name: "no sink type", src: "failure: skip\n", wantMsg: "must define exactly one of"},
		{name: "two sink types", src: "http:\n  url: a\nkafka:\n  url: b\n", wantMsg: "must define exactly one of"},
		{name: "kafka invalid", src: "kafka:\n  topic: x\n"},
		{name: "nats-core invalid", src: "nats-core:\n  name: x\n"},
		{name: "nats-jetstream invalid", src: "nats-jetstream:\n  url: nats://x\n"},
		{name: "http invalid", src: "http:\n  method: GET\n"},
		{name: "rabbitmq invalid", src: "rabbitmq:\n  routingKey: x\n"},
		{name: "rabbitmq-stream invalid", src: "rabbitmq-stream:\n  payload: x\n"},
		{name: "unknown sibling field", src: "http:\n  url: a\nextra: 1\n"},
		{name: "invalid operation", src: "http:\n  url: a\noperations: [upsert]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(tt.src), &cfg)
			if err == nil {
				t.Fatalf("expected error for:\n%s", tt.src)
			}
			if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("error = %q, want it to contain %q", err, tt.wantMsg)
			}
		})
	}
}

func TestConfigMarshalYAML(t *testing.T) {
	cfg := Config{
		Sink:       &HTTPSink{URL: mustTemplate(t, "https://example.com")},
		Failure:    &resilience.Failure{Mode: pkgresilience.FailureModeSkip},
		Retry:      &resilience.Retry{MaxAttempts: 2},
		Operations: []domain.OperationType{domain.OperationTypeInsert},
	}

	out, err := cfg.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	m := out.(map[string]any)
	for _, key := range []string{"http", "failure", "retry", "operations"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q in %v", key, m)
		}
	}
}

func TestConfigMarshalYAMLEmpty(t *testing.T) {
	out, err := Config{}.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if m := out.(map[string]any); len(m) != 0 {
		t.Fatalf("MarshalYAML = %v, want empty map", m)
	}
}

func TestCrossValidate(t *testing.T) {
	stream := func(subject string) Config {
		return Config{Sink: &NatsJetStreamSink{
			NatsCommon: NatsCommon{URL: mustTemplate(t, "nats://x")},
			Stream:     NatsStream{Name: "S", Subjects: []string{subject}},
		}}
	}

	if err := CrossValidate([][]Config{{stream("a")}, {stream("a")}}); err != nil {
		t.Fatalf("CrossValidate(agreeing) = %v, want nil", err)
	}

	err := CrossValidate([][]Config{{stream("a")}, {stream("b")}})
	if !errs.Is(err, errors.ErrCrossValidate) {
		t.Fatalf("CrossValidate(divergent) = %v, want ErrCrossValidate", err)
	}
}

// unmarshalConfig decodes src into a Config, failing the test on error.
func unmarshalConfig(t *testing.T, src string) Config {
	t.Helper()
	var cfg Config
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("Unmarshal:\n%s\nerr: %v", src, err)
	}
	return cfg
}

package config

import (
	_ "embed"
	"fmt"

	"hump/internal/types/sink"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

var (
	//go:embed templ/kafka_topic.tmpl
	defaultKafkaTopic string

	//go:embed templ/client_id.tmpl
	defaultKafkaClientID string

	//go:embed templ/payload.tmpl
	defaultKafkaPayload string

	//go:embed templ/idempotency_header.tmpl
	defaultKafkaIdempotencyKey string

	//go:embed templ/traceparent_header.tmpl
	defaultKafkaTraceparent string
)

type KafkaSink struct {
	URL *template.Template `yaml:"url"                  validate:"required"`

	TLS      bool               `yaml:"tls,omitempty"`
	SASL     string             `yaml:"sasl,omitempty"`
	Username *template.Template `yaml:"username,omitempty"`
	Password *template.Template `yaml:"password,omitempty"`

	ClientID *template.Template `yaml:"clientId,omitempty"`

	Topic        *template.Template            `yaml:"topic,omitempty"`
	PartitionKey *template.Template            `yaml:"partitionKey,omitempty"`
	Payload      *template.Template            `yaml:"payload,omitempty"`
	Headers      map[string]*template.Template `yaml:"headers,omitempty"`
}

func (*KafkaSink) Type() sink.SinkType { return sink.SinkTypeKafka }

func (s *KafkaSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sink.SinkTypeKafka)
	}

	return fmt.Sprintf("%s(%s)", string(sink.SinkTypeKafka), s.URL.Source())
}

// Metadata returns the connection host (rendered broker URL) and the rendered
// topic for the given context.
func (s *KafkaSink) Metadata(context sink.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Topic.Render(context)

	return parseHost(url), target
}

// CrossValidate is a no-op for Kafka sinks; they have no cross-sink invariants.
func (*KafkaSink) CrossValidate([]Tables) error { return nil }

func (k *KafkaSink) UnmarshalYAML(node ast.Node) error {
	type kafka KafkaSink
	if err := decode(node, (*kafka)(k)); err != nil {
		return err
	}

	k.setDefaults()

	return nil
}

func (k *KafkaSink) setDefaults() {
	if k.Topic == nil {
		t, _ := template.New(defaultKafkaTopic)
		k.Topic = t
	}

	if k.Payload == nil {
		t, _ := template.New(defaultKafkaPayload)
		k.Payload = t
	}

	if k.ClientID == nil {
		t, _ := template.New(defaultKafkaClientID)
		k.ClientID = t
	}

	if len(k.Headers) == 0 {
		k.Headers = map[string]*template.Template{}

		idempotency, _ := template.New(defaultKafkaIdempotencyKey)
		k.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(defaultKafkaTraceparent)
		k.Headers["traceparent"] = traceparent
	}
}

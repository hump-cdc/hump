package sink

import (
	_ "embed"
	"fmt"

	"hump/config/decoder"
	sinkport "hump/internal/port/sink"
	"hump/pkg/metadata"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

var (
	//go:embed templ/kafka_topic.tmpl
	defaultKafkaTopic string
)

// KafkaSink publishes each change as a Kafka record.
type KafkaSink struct {
	URL *template.Template `yaml:"url" validate:"required"`

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

// Type returns the sink type key.
func (*KafkaSink) Type() sinkport.Type { return sinkport.SinkTypeKafka }

// DisplayName returns the sink type followed by its URL template.
func (s *KafkaSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeKafka)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeKafka), s.URL.Source())
}

// Metadata returns the host of the rendered URL and the rendered topic.
func (s *KafkaSink) Metadata(context sinkport.Context) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Topic.Render(context)

	return metadata.ParseHost(url), target
}

// CrossValidate accepts any configuration: Kafka sinks share no state across tables.
func (*KafkaSink) CrossValidate([][]sinkport.Definition) error { return nil }

// UnmarshalYAML decodes the sink strictly and fills in its defaults.
func (s *KafkaSink) UnmarshalYAML(node ast.Node) error {
	type kafka KafkaSink
	if err := decoder.Decode(node, (*kafka)(s)); err != nil {
		return err
	}

	s.setDefaults()

	return nil
}

func (s *KafkaSink) setDefaults() {
	if s.Topic == nil {
		t, _ := template.New(defaultKafkaTopic)
		s.Topic = t
	}

	if s.Payload == nil {
		t, _ := template.New(DefaultPayload)
		s.Payload = t
	}

	if s.ClientID == nil {
		t, _ := template.New(DefaultClientID)
		s.ClientID = t
	}

	if len(s.Headers) == 0 {
		s.Headers = map[string]*template.Template{}

		idempotency, _ := template.New(DefaultIdempotencyKey)
		s.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(DefaultTraceparent)
		s.Headers["traceparent"] = traceparent
	}
}

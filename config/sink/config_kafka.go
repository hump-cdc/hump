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

func (*KafkaSink) Type() sinkport.SinkType { return sinkport.SinkTypeKafka }

func (s *KafkaSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeKafka)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeKafka), s.URL.Source())
}

func (s *KafkaSink) Metadata(context sinkport.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Topic.Render(context)

	return metadata.ParseHost(url), target
}

func (*KafkaSink) CrossValidate([][]sinkport.Definition) error { return nil }

func (k *KafkaSink) UnmarshalYAML(node ast.Node) error {
	type kafka KafkaSink
	if err := decoder.Decode(node, (*kafka)(k)); err != nil {
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
		t, _ := template.New(DefaultPayload)
		k.Payload = t
	}

	if k.ClientID == nil {
		t, _ := template.New(DefaultClientID)
		k.ClientID = t
	}

	if len(k.Headers) == 0 {
		k.Headers = map[string]*template.Template{}

		idempotency, _ := template.New(DefaultIdempotencyKey)
		k.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(DefaultTraceparent)
		k.Headers["traceparent"] = traceparent
	}
}

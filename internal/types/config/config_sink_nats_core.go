package config

import (
	"fmt"

	"hump/internal/types/sink"

	"github.com/goccy/go-yaml/ast"
)

// NatsCoreSink is the config for a Core NATS sink.
type NatsCoreSink struct {
	NatsCommon `yaml:",inline"`
}

func (*NatsCoreSink) Type() sink.SinkType { return sink.SinkTypeNatsCore }

func (s *NatsCoreSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sink.SinkTypeNatsCore)
	}

	return fmt.Sprintf("%s(%s)", string(sink.SinkTypeNatsCore), s.URL.Source())
}

// Metadata returns the connection host (rendered URL) and the rendered subject
// for the given context.
func (s *NatsCoreSink) Metadata(context sink.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Subject.Render(context)

	return parseHost(url), target
}

func (s *NatsCoreSink) UnmarshalYAML(node ast.Node) error {
	type natsCore NatsCoreSink
	if err := decode(node, (*natsCore)(s)); err != nil {
		return err
	}

	s.NatsCommon.setDefaults()

	return nil
}

// CrossValidate is a no-op for Core NATS sinks; they have no cross-sink
// invariants.
func (*NatsCoreSink) CrossValidate([]Tables) error { return nil }

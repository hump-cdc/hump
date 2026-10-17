package sink

import (
	"fmt"

	"hump/config/decoder"
	sinkport "hump/internal/port/sink"
	"hump/pkg/metadata"

	"github.com/goccy/go-yaml/ast"
)

// NatsCoreSink publishes each change as a core NATS message.
type NatsCoreSink struct {
	NatsCommon `yaml:",inline"`
}

// Type returns the sink type key.
func (*NatsCoreSink) Type() sinkport.Type { return sinkport.SinkTypeNatsCore }

// DisplayName returns the sink type followed by its URL template.
func (s *NatsCoreSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeNatsCore)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeNatsCore), s.URL.Source())
}

// Metadata returns the host of the rendered URL and the rendered subject.
func (s *NatsCoreSink) Metadata(context sinkport.Context) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Subject.Render(context)

	return metadata.ParseHost(url), target
}

// UnmarshalYAML decodes the sink strictly and fills in its defaults.
func (s *NatsCoreSink) UnmarshalYAML(node ast.Node) error {
	type natsCore NatsCoreSink
	if err := decoder.Decode(node, (*natsCore)(s)); err != nil {
		return err
	}

	s.setDefaults()

	return nil
}

// CrossValidate accepts any configuration: core NATS sinks share no state across tables.
func (*NatsCoreSink) CrossValidate([][]sinkport.Definition) error { return nil }

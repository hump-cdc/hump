package sink

import (
	"fmt"

	"hump/config/decoder"
	sinkport "hump/internal/port/sink"
	"hump/pkg/metadata"

	"github.com/goccy/go-yaml/ast"
)

type NatsCoreSink struct {
	NatsCommon `yaml:",inline"`
}

func (*NatsCoreSink) Type() sinkport.SinkType { return sinkport.SinkTypeNatsCore }

func (s *NatsCoreSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeNatsCore)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeNatsCore), s.URL.Source())
}

func (s *NatsCoreSink) Metadata(context sinkport.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Subject.Render(context)

	return metadata.ParseHost(url), target
}

func (s *NatsCoreSink) UnmarshalYAML(node ast.Node) error {
	type natsCore NatsCoreSink
	if err := decoder.Decode(node, (*natsCore)(s)); err != nil {
		return err
	}

	s.NatsCommon.setDefaults()

	return nil
}

func (*NatsCoreSink) CrossValidate([][]sinkport.Definition) error { return nil }

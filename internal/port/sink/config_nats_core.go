package sink

import (
	"fmt"

	"hump/pkg/config"
	"hump/pkg/metadata"

	"github.com/goccy/go-yaml/ast"
)

type NatsCoreSink struct {
	NatsCommon `yaml:",inline"`
}

func (*NatsCoreSink) Type() SinkType { return SinkTypeNatsCore }

func (s *NatsCoreSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(SinkTypeNatsCore)
	}

	return fmt.Sprintf("%s(%s)", string(SinkTypeNatsCore), s.URL.Source())
}

func (s *NatsCoreSink) Metadata(context SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Subject.Render(context)

	return metadata.ParseHost(url), target
}

func (s *NatsCoreSink) UnmarshalYAML(node ast.Node) error {
	type natsCore NatsCoreSink
	if err := config.Decode(node, (*natsCore)(s)); err != nil {
		return err
	}

	s.NatsCommon.setDefaults()

	return nil
}

func (*NatsCoreSink) CrossValidate([][]Definition) error { return nil }

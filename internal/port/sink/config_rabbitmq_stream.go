package sink

import (
	_ "embed"
	"fmt"
	"time"

	"hump/internal/types/errs"
	"hump/pkg/config"
	"hump/pkg/metadata"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

var (
	//go:embed templ/rabbitmq_stream_name.tmpl
	defaultRabbitMQStreamName string
)

// RabbitMQStreamMode selects how a RabbitMQ Streams stream is reconciled at
// first publish.
type RabbitMQStreamMode string

const (
	RabbitMQStreamModePassive           RabbitMQStreamMode = "passive"
	RabbitMQStreamModeCreateWhenMissing RabbitMQStreamMode = "create-when-missing"
)

type RabbitMQStreamConfig struct {
	Name                *template.Template `yaml:"name,omitempty"`
	Mode                RabbitMQStreamMode `yaml:"mode,omitempty"                validate:"omitempty,oneof=passive create-when-missing"`
	MaxAge              time.Duration      `yaml:"maxAge,omitempty"              validate:"omitempty,gt=0"`
	MaxLengthBytes      int64              `yaml:"maxLengthBytes,omitempty"      validate:"omitempty,gt=0"`
	MaxSegmentSizeBytes int64              `yaml:"maxSegmentSizeBytes,omitempty" validate:"omitempty,gt=0"`
}

type RabbitMQStreamSink struct {
	RabbitMQCommon `yaml:",inline"`

	Stream RabbitMQStreamConfig `yaml:"stream,omitempty"`
}

func (*RabbitMQStreamSink) Type() SinkType { return SinkTypeRabbitMQStream }

func (s *RabbitMQStreamSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(SinkTypeRabbitMQStream)
	}

	return fmt.Sprintf("%s(%s)", string(SinkTypeRabbitMQStream), s.URL.Source())
}

func (s *RabbitMQStreamSink) Metadata(context SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Stream.Name.Render(context)

	return metadata.ParseHost(url), target
}

func (s *RabbitMQStreamSink) UnmarshalYAML(node ast.Node) error {
	type rabbitmqStream RabbitMQStreamSink
	if err := config.Decode(node, (*rabbitmqStream)(s)); err != nil {
		return err
	}

	s.RabbitMQCommon.SetDefaults()
	s.setDefaults()

	return nil
}

func (s *RabbitMQStreamSink) setDefaults() {
	if s.Stream.Name == nil {
		t, _ := template.New(defaultRabbitMQStreamName)
		s.Stream.Name = t
	}
	if s.Stream.Mode == "" {
		s.Stream.Mode = RabbitMQStreamModeCreateWhenMissing
	}
	if s.Stream.MaxAge == 0 {
		s.Stream.MaxAge = 24 * time.Hour
	}
	if s.Stream.MaxLengthBytes == 0 {
		s.Stream.MaxLengthBytes = 2 * 1024 * 1024 * 1024 // 2 GiB
	}
	if s.Stream.MaxSegmentSizeBytes == 0 {
		s.Stream.MaxSegmentSizeBytes = 500 * 1024 * 1024 // 500 MiB
	}
}

func (*RabbitMQStreamSink) CrossValidate(groups [][]Definition) error {
	type rmqStreamEntry struct {
		maxAge              time.Duration
		maxLengthBytes      int64
		maxSegmentSizeBytes int64
	}
	seen := make(map[string]rmqStreamEntry)

	for _, group := range groups {
		for _, sc := range group {
			rs, ok := sc.(*RabbitMQStreamSink)
			if !ok || rs == nil {
				continue
			}

			url := ""
			if rs.URL != nil {
				url = rs.URL.Source()
			}
			streamName := ""
			if rs.Stream.Name != nil {
				streamName = rs.Stream.Name.Source()
			}
			key := fmt.Sprintf("%s\x00%s", url, streamName)

			e := rmqStreamEntry{
				maxAge:              rs.Stream.MaxAge,
				maxLengthBytes:      rs.Stream.MaxLengthBytes,
				maxSegmentSizeBytes: rs.Stream.MaxSegmentSizeBytes,
			}

			prev, exists := seen[key]
			if !exists {
				seen[key] = e
				continue
			}

			if prev != e {
				return fmt.Errorf("%w: stream %q: divergent maxAge/maxLengthBytes/maxSegmentSizeBytes", errs.ErrCrossValidate, streamName)
			}
		}
	}

	return nil
}

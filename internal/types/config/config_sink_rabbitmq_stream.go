package config

import (
	_ "embed"
	"fmt"
	"time"

	"hump/internal/template"
	"hump/internal/types/errs"
	"hump/internal/types/sink"

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

// RabbitMQStreamConfig holds the stream declaration knobs.
type RabbitMQStreamConfig struct {
	Name                *template.Template `yaml:"name,omitempty"`
	Mode                RabbitMQStreamMode `yaml:"mode,omitempty"                validate:"omitempty,oneof=passive create-when-missing"`
	MaxAge              time.Duration      `yaml:"maxAge,omitempty"              validate:"omitempty,gt=0"`
	MaxLengthBytes      int64              `yaml:"maxLengthBytes,omitempty"      validate:"omitempty,gt=0"`
	MaxSegmentSizeBytes int64              `yaml:"maxSegmentSizeBytes,omitempty" validate:"omitempty,gt=0"`
}

// RabbitMQStreamSink is the config for a RabbitMQ Streams sink.
type RabbitMQStreamSink struct {
	RabbitMQCommon `yaml:",inline"`

	Stream RabbitMQStreamConfig `yaml:"stream,omitempty"`
}

func (*RabbitMQStreamSink) Type() sink.SinkType { return sink.SinkTypeRabbitMQStream }

func (s *RabbitMQStreamSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sink.SinkTypeRabbitMQStream)
	}

	return fmt.Sprintf("%s(%s)", string(sink.SinkTypeRabbitMQStream), s.URL.Source())
}

func (s *RabbitMQStreamSink) UnmarshalYAML(node ast.Node) error {
	type rabbitmqStream RabbitMQStreamSink
	if err := decode(node, (*rabbitmqStream)(s)); err != nil {
		return err
	}

	s.RabbitMQCommon.setDefaults()
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

// CrossValidate returns an error wrapping [errs.ErrCrossValidate] on the
// first (URL, stream-name) pair whose declared retention parameters differ
// between two RabbitMQ Stream sinks.
//
// The receiver is not inspected — the method's result depends only on tables.
func (*RabbitMQStreamSink) CrossValidate(tables []Tables) error {
	type rmqStreamEntry struct {
		maxAge              time.Duration
		maxLengthBytes      int64
		maxSegmentSizeBytes int64
	}
	seen := make(map[string]rmqStreamEntry)

	for _, table := range tables {
		for _, sc := range table.Sinks {
			rs, ok := sc.Sink.(*RabbitMQStreamSink)
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

package sink

import (
	_ "embed"
	"fmt"
	"time"

	"hump/config/decoder"
	"hump/config/errors"
	sinkport "hump/internal/port/sink"
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
	// RabbitMQStreamModePassive asserts the stream already exists and never
	// declares it.
	RabbitMQStreamModePassive RabbitMQStreamMode = "passive"

	// RabbitMQStreamModeCreateWhenMissing declares the stream on first publish if
	// it is absent.
	RabbitMQStreamModeCreateWhenMissing RabbitMQStreamMode = "create-when-missing"
)

// RabbitMQStreamConfig is the stream a sink publishes to, how it is declared,
// and its retention limits.
type RabbitMQStreamConfig struct {
	Name                *template.Template `yaml:"name,omitempty"`
	Mode                RabbitMQStreamMode `yaml:"mode,omitempty"                validate:"omitempty,oneof=passive create-when-missing"`
	MaxAge              time.Duration      `yaml:"maxAge,omitempty"              validate:"omitempty,gt=0"`
	MaxLengthBytes      int64              `yaml:"maxLengthBytes,omitempty"      validate:"omitempty,gt=0"`
	MaxSegmentSizeBytes int64              `yaml:"maxSegmentSizeBytes,omitempty" validate:"omitempty,gt=0"`
}

// RabbitMQStreamSink publishes each change to a RabbitMQ Streams stream.
type RabbitMQStreamSink struct {
	RabbitMQCommon `yaml:",inline"`

	Stream RabbitMQStreamConfig `yaml:"stream,omitempty"`
}

// Type returns the sink type key.
func (*RabbitMQStreamSink) Type() sinkport.Type { return sinkport.SinkTypeRabbitMQStream }

// DisplayName returns the sink type followed by its URL template.
func (s *RabbitMQStreamSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeRabbitMQStream)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeRabbitMQStream), s.URL.Source())
}

// Metadata returns the host of the rendered URL and the rendered stream name.
func (s *RabbitMQStreamSink) Metadata(context sinkport.Context) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Stream.Name.Render(context)

	return metadata.ParseHost(url), target
}

// UnmarshalYAML decodes the sink strictly and fills in its defaults.
func (s *RabbitMQStreamSink) UnmarshalYAML(node ast.Node) error {
	type rabbitmqStream RabbitMQStreamSink
	if err := decoder.Decode(node, (*rabbitmqStream)(s)); err != nil {
		return err
	}

	s.SetDefaults()
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

// CrossValidate rejects stream sinks that share a URL and stream but differ in
// their retention limits.
func (*RabbitMQStreamSink) CrossValidate(groups [][]sinkport.Definition) error {
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
				return fmt.Errorf("%w: stream %q: divergent maxAge/maxLengthBytes/maxSegmentSizeBytes", errors.ErrCrossValidate, streamName)
			}
		}
	}

	return nil
}

package sink

import (
	"fmt"
	"slices"
	"sort"

	"hump/config/decoder"
	"hump/config/errors"
	sinkport "hump/internal/port/sink"
	"hump/pkg/metadata"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

// JetStreamReconcileMode selects how a JetStream stream is reconciled at first
// publish.
type JetStreamReconcileMode string

const (
	// JetStreamReconcileCreateOrUpdate creates the stream or updates an existing
	// one to match the configuration.
	JetStreamReconcileCreateOrUpdate JetStreamReconcileMode = "create-or-update"

	// JetStreamReconcileCreateWhenMissing creates the stream only if it is absent
	// and leaves an existing one untouched.
	JetStreamReconcileCreateWhenMissing JetStreamReconcileMode = "create-when-missing"
)

// NatsStream names the JetStream stream a sink publishes to and the subjects
// it captures.
type NatsStream struct {
	Name     string   `yaml:"name"               validate:"required"`
	Subjects []string `yaml:"subjects,omitempty"`
}

// NatsJetStreamSink publishes each change to a JetStream stream, which it
// reconciles before the first publish.
type NatsJetStreamSink struct {
	NatsCommon `yaml:",inline"`

	Stream NatsStream `yaml:"stream"                 validate:"required"`

	MsgID *template.Template `yaml:"msgId,omitempty"`

	Mode      JetStreamReconcileMode `yaml:"mode,omitempty"      validate:"omitempty,oneof=create-or-update create-when-missing"`
	Retention string                 `yaml:"retention,omitempty" validate:"omitempty,oneof=limits interest workqueue"`
	Storage   string                 `yaml:"storage,omitempty"   validate:"omitempty,oneof=file memory"`
}

// Type returns the sink type key.
func (*NatsJetStreamSink) Type() sinkport.Type { return sinkport.SinkTypeNatsJetStream }

// DisplayName returns the sink type followed by its URL template.
func (s *NatsJetStreamSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeNatsJetStream)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeNatsJetStream), s.URL.Source())
}

// Metadata returns the host of the rendered URL and the rendered subject.
func (s *NatsJetStreamSink) Metadata(context sinkport.Context) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Subject.Render(context)

	return metadata.ParseHost(url), target
}

// UnmarshalYAML decodes the sink strictly and fills in its defaults.
func (s *NatsJetStreamSink) UnmarshalYAML(node ast.Node) error {
	type natsJetStream NatsJetStreamSink
	if err := decoder.Decode(node, (*natsJetStream)(s)); err != nil {
		return err
	}

	s.NatsCommon.setDefaults()
	s.setDefaults()

	return nil
}

func (s *NatsJetStreamSink) setDefaults() {
	if s.Mode == "" {
		s.Mode = JetStreamReconcileCreateOrUpdate
	}
	if s.MsgID == nil {
		t, _ := template.New(DefaultIdempotencyKey)
		s.MsgID = t
	}
}

// CrossValidate rejects JetStream sinks that share a URL and stream but differ
// in subjects, retention or storage.
func (*NatsJetStreamSink) CrossValidate(groups [][]sinkport.Definition) error {
	type jsEntry struct {
		subjects  []string
		retention string
		storage   string
	}
	seen := make(map[string]jsEntry)

	for _, group := range groups {
		for _, sc := range group {
			// Find JetStream sinks.
			jsSink, ok := sc.(*NatsJetStreamSink)
			if !ok || jsSink == nil {
				continue
			}

			// Generate unique key for the sink.
			url := ""
			if jsSink.URL != nil {
				url = jsSink.URL.Source()
			}
			key := fmt.Sprintf("%s\x00%s", url, jsSink.Stream.Name)

			// Create sorted subjects.
			subjects := slices.Clone(jsSink.Stream.Subjects)
			sort.Strings(subjects)

			// Validate JetStream sink configuration against previously seen sinks with the same (URL, stream).
			e := jsEntry{
				subjects:  subjects,
				retention: jsSink.Retention,
				storage:   jsSink.Storage,
			}

			prev, exists := seen[key]
			if !exists {
				seen[key] = e
				continue
			}

			if !slices.Equal(prev.subjects, e.subjects) ||
				prev.retention != e.retention ||
				prev.storage != e.storage {
				return fmt.Errorf("%w: stream %q: divergent subjects/retention/storage", errors.ErrCrossValidate, jsSink.Stream.Name)
			}
		}
	}

	return nil
}

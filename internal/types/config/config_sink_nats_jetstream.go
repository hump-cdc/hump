package config

import (
	"fmt"
	"slices"
	"sort"

	"hump/internal/template"
	"hump/internal/types/errs"
	"hump/internal/types/sink"

	"github.com/goccy/go-yaml/ast"
)

// JetStreamReconcileMode selects how a JetStream stream is reconciled at
// first publish.
type JetStreamReconcileMode string

const (
	JetStreamReconcileCreateOrUpdate    JetStreamReconcileMode = "create-or-update"
	JetStreamReconcileCreateWhenMissing JetStreamReconcileMode = "create-when-missing"
)

// NatsStream names a JetStream stream and the subjects it captures.
type NatsStream struct {
	Name     string   `yaml:"name"               validate:"required"`
	Subjects []string `yaml:"subjects,omitempty"`
}

// NatsJetStreamSink is the config for a JetStream sink.
type NatsJetStreamSink struct {
	NatsCommon `yaml:",inline"`

	Stream NatsStream `yaml:"stream"                 validate:"required"`

	MsgID *template.Template `yaml:"msgId,omitempty"`

	Mode      JetStreamReconcileMode `yaml:"mode,omitempty"      validate:"omitempty,oneof=create-or-update create-when-missing"`
	Retention string                 `yaml:"retention,omitempty" validate:"omitempty,oneof=limits interest workqueue"`
	Storage   string                 `yaml:"storage,omitempty"   validate:"omitempty,oneof=file memory"`
}

func (*NatsJetStreamSink) Type() sink.SinkType { return sink.SinkTypeNatsJetStream }

func (s *NatsJetStreamSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sink.SinkTypeNatsJetStream)
	}

	return fmt.Sprintf("%s(%s)", string(sink.SinkTypeNatsJetStream), s.URL.Source())
}

// Metadata returns the connection host (rendered URL) and the rendered subject
// for the given context.
func (s *NatsJetStreamSink) Metadata(context sink.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Subject.Render(context)

	return parseHost(url), target
}

func (s *NatsJetStreamSink) UnmarshalYAML(node ast.Node) error {
	type natsJetStream NatsJetStreamSink
	if err := decode(node, (*natsJetStream)(s)); err != nil {
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
		t, _ := template.New(defaultNatsIdempotencyKey)
		s.MsgID = t
	}
}

// CrossValidate returns an error wrapping [errs.ErrCrossValidate] on the
// first (URL, stream) pair whose declared subjects, retention, or storage
// differ between two JetStream sinks.
//
// The receiver is not inspected — the method's result depends only on tables.
func (*NatsJetStreamSink) CrossValidate(tables []Tables) error {
	type jsEntry struct {
		subjects  []string
		retention string
		storage   string
	}
	seen := make(map[string]jsEntry)

	for _, table := range tables {
		for _, sink := range table.Sinks {
			// Find JetStream sinks.
			jsSink, ok := sink.Sink.(*NatsJetStreamSink)
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
				return fmt.Errorf("%w: stream %q: divergent subjects/retention/storage", errs.ErrCrossValidate, jsSink.Stream.Name)
			}
		}
	}

	return nil
}

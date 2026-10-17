// Package sink holds the configuration of a table's sinks: one type per sink
// kind, their defaults, and the validation across tables.
package sink

import (
	"fmt"
	"slices"

	"hump/config/decoder"
	"hump/config/resilience"
	"hump/internal/domain"
	sinkport "hump/internal/port/sink"

	"github.com/goccy/go-yaml/ast"

	_ "embed"
)

// Default templates shared by the sink types, embedded from templ/.
var (
	//go:embed templ/payload.tmpl
	DefaultPayload string

	//go:embed templ/client_id.tmpl
	DefaultClientID string

	//go:embed templ/idempotency_header.tmpl
	DefaultIdempotencyKey string

	//go:embed templ/traceparent_header.tmpl
	DefaultTraceparent string
)

// Config is one sink entry of a table: the sink definition plus its delivery
// policies and the operations it receives.
type Config struct {
	Sink sinkport.Definition `yaml:"-"`

	Failure *resilience.Failure `yaml:"failure,omitempty"`
	Retry   *resilience.Retry   `yaml:"retry,omitempty"`

	Operations []domain.OperationType `yaml:"operations,omitempty" validate:"omitempty,min=1,unique,dive,oneof=insert update delete truncate"`
}

// MarshalYAML writes the definition back under its sink type key, next to the
// policies and operations.
func (sc Config) MarshalYAML() (any, error) {
	out := map[string]any{}

	if sc.Sink != nil {
		out[string(sc.Sink.Type())] = sc.Sink
	}

	if sc.Failure != nil {
		out["failure"] = sc.Failure
	}

	if sc.Retry != nil {
		out["retry"] = sc.Retry
	}

	if len(sc.Operations) > 0 {
		out["operations"] = sc.Operations
	}

	return out, nil
}

// UnmarshalYAML decodes the sink definition from its type key, then the rest
// of the entry, and fills in the defaults.
func (sc *Config) UnmarshalYAML(node ast.Node) error {
	mapping, ok := node.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("line %d: sink: must be a mapping", decoder.NodeLine(node))
	}

	s, err := unmarshalSink(mapping)
	if err != nil {
		return err
	}
	sc.Sink = s

	stripped := nodeWithoutSinkTypes(mapping)

	type sinkConfig Config
	if err := decoder.Decode(stripped, (*sinkConfig)(sc)); err != nil {
		return err
	}

	sc.setDefaults()

	return nil
}

func (sc *Config) setDefaults() {
	if sc.Failure == nil {
		sc.Failure = &resilience.Failure{}
	}
	sc.Failure.SetDefaults()

	if sc.Retry == nil {
		sc.Retry = &resilience.Retry{}
	}
	sc.Retry.SetDefaults()

	if len(sc.Operations) == 0 {
		sc.Operations = slices.Clone(domain.OperationTypes)
	}
}

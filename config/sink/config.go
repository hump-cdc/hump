package sink

import (
	"fmt"

	"hump/config/decoder"
	"hump/config/resilience"
	sinkport "hump/internal/port/sink"

	"github.com/goccy/go-yaml/ast"

	_ "embed"
)

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

type Config struct {
	Sink sinkport.Definition `yaml:"-"`

	Failure *resilience.Failure `yaml:"failure,omitempty"`
	Retry   *resilience.Retry   `yaml:"retry,omitempty"`
}

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

	return out, nil
}

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
}

package config

import (
	"fmt"

	"hump/internal/types/sink"

	"github.com/goccy/go-yaml/ast"
)

type SinkConfig struct {
	Sink sink.Sink `yaml:"-"`

	Failure  *Failure  `yaml:"failure,omitempty"`
	Retry    *Retry    `yaml:"retry,omitempty"`
	Delivery *Delivery `yaml:"delivery,omitempty"`
}

func (sc SinkConfig) MarshalYAML() (any, error) {
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

	if sc.Delivery != nil {
		out["delivery"] = sc.Delivery
	}

	return out, nil
}

func (sc *SinkConfig) UnmarshalYAML(node ast.Node) error {
	mapping, ok := node.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("line %d: sink: must be a mapping", nodeLine(node))
	}

	s, err := unmarshalSink(mapping)
	if err != nil {
		return err
	}
	sc.Sink = s

	stripped := nodeWithoutSinkTypes(mapping)

	type sink SinkConfig
	if err := decode(stripped, (*sink)(sc)); err != nil {
		return err
	}

	sc.setDefaults()

	return nil
}

// nodeWithoutSinkTypes returns a shallow clone of m with any entry
// whose key matches a known SinkType removed. The original node is left intact.
func nodeWithoutSinkTypes(m *ast.MappingNode) *ast.MappingNode {
	skip := make(map[string]struct{}, len(sink.SinkTypes))
	for _, t := range sink.SinkTypes {
		skip[string(t)] = struct{}{}
	}

	filtered := make([]*ast.MappingValueNode, 0, len(m.Values))
	for _, kv := range m.Values {
		if _, drop := skip[kv.Key.String()]; drop {
			continue
		}
		filtered = append(filtered, kv)
	}

	clone := *m
	clone.Values = filtered

	return &clone
}

func (sc *SinkConfig) setDefaults() {
	if sc.Failure == nil {
		sc.Failure = &Failure{}
	}
	sc.Failure.setDefaults()

	if sc.Retry == nil {
		sc.Retry = &Retry{}
	}
	sc.Retry.setDefaults()

	if sc.Delivery == nil {
		sc.Delivery = &Delivery{}
	}
	sc.Delivery.setDefaults()
}

package config

import (
	"fmt"

	"hump/internal/types/sink"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// unmarshalSink decodes the http/kafka discriminator key from a sink mapping
// into the matching concrete Sink. Errors when zero or multiple discriminator
// keys are present.
func unmarshalSink(node *ast.MappingNode) (sink.Sink, error) {
	t, value := getSinkType(node)

	switch t {
	case sink.SinkTypeKafka:
		var k KafkaSink
		if err := yaml.NodeToValue(value, &k, yaml.Strict()); err != nil {
			return nil, err
		}
		return &k, nil

	case sink.SinkTypeNatsCore:
		var n NatsCoreSink
		if err := yaml.NodeToValue(value, &n, yaml.Strict()); err != nil {
			return nil, err
		}
		return &n, nil

	case sink.SinkTypeNatsJetStream:
		var n NatsJetStreamSink
		if err := yaml.NodeToValue(value, &n, yaml.Strict()); err != nil {
			return nil, err
		}
		return &n, nil

	case sink.SinkTypeHTTP:
		var h HTTPSink
		if err := yaml.NodeToValue(value, &h, yaml.Strict()); err != nil {
			return nil, err
		}
		return &h, nil

	case sink.SinkTypeRabbitMQ:
		var r RabbitMQSink
		if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil {
			return nil, err
		}
		return &r, nil

	case sink.SinkTypeRabbitMQStream:
		var r RabbitMQStreamSink
		if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil {
			return nil, err
		}
		return &r, nil

	default:
		return nil, fmt.Errorf("line %d: sink: must define exactly one of %v", nodeLine(node), sink.SinkTypes)
	}
}

// getSinkType returns the single sink discriminator present in node along with
// its value node. Returns ("", nil) if none or more than one is present.
func getSinkType(node *ast.MappingNode) (sink.SinkType, ast.Node) {
	var foundType sink.SinkType
	var foundValue ast.Node

	for _, t := range sink.SinkTypes {
		v := getSinkKey(node, string(t))
		if v == nil {
			continue
		}

		if foundType != "" {
			return "", nil
		}
		foundType = t
		foundValue = v
	}

	return foundType, foundValue
}

// getSinkKey returns the value node for key in a YAML mapping, or nil if the
// key is absent.
func getSinkKey(node *ast.MappingNode, key string) ast.Node {
	for _, kv := range node.Values {
		if kv.Key.String() == key {
			return kv.Value
		}
	}

	return nil
}

// nodeLine returns the source line of an AST node, or 0 if unavailable.
func nodeLine(node ast.Node) int {
	if node == nil {
		return 0
	}
	tok := node.GetToken()
	if tok == nil || tok.Position == nil {
		return 0
	}
	return tok.Position.Line
}

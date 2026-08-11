package sink

import (
	"fmt"

	"hump/pkg/config"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

func unmarshalSink(node *ast.MappingNode) (Definition, error) {
	t, value := getSinkType(node)

	switch t {
	case SinkTypeKafka:
		var k KafkaSink
		if err := yaml.NodeToValue(value, &k, yaml.Strict()); err != nil {
			return nil, err
		}
		return &k, nil

	case SinkTypeNatsCore:
		var n NatsCoreSink
		if err := yaml.NodeToValue(value, &n, yaml.Strict()); err != nil {
			return nil, err
		}
		return &n, nil

	case SinkTypeNatsJetStream:
		var n NatsJetStreamSink
		if err := yaml.NodeToValue(value, &n, yaml.Strict()); err != nil {
			return nil, err
		}
		return &n, nil

	case SinkTypeHTTP:
		var h HTTPSink
		if err := yaml.NodeToValue(value, &h, yaml.Strict()); err != nil {
			return nil, err
		}
		return &h, nil

	case SinkTypeRabbitMQ:
		var r RabbitMQSink
		if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil {
			return nil, err
		}
		return &r, nil

	case SinkTypeRabbitMQStream:
		var r RabbitMQStreamSink
		if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil {
			return nil, err
		}
		return &r, nil

	default:
		return nil, fmt.Errorf("line %d: sink: must define exactly one of %v", config.NodeLine(node), SinkTypes)
	}
}

var crossValidators = []CrossValidator{
	(*HTTPSink)(nil),
	(*KafkaSink)(nil),
	(*NatsCoreSink)(nil),
	(*NatsJetStreamSink)(nil),
	(*RabbitMQSink)(nil),
	(*RabbitMQStreamSink)(nil),
}

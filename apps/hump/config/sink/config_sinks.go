package sink

import (
	"fmt"

	"hump/config/decoder"
	sinkport "hump/internal/port/sink"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

func unmarshalSink(node *ast.MappingNode) (sinkport.Definition, error) {
	t, value := getSinkType(node)

	switch t {
	case sinkport.SinkTypeKafka:
		var k KafkaSink
		if err := yaml.NodeToValue(value, &k, yaml.Strict()); err != nil {
			return nil, err
		}
		return &k, nil

	case sinkport.SinkTypeNatsCore:
		var n NatsCoreSink
		if err := yaml.NodeToValue(value, &n, yaml.Strict()); err != nil {
			return nil, err
		}
		return &n, nil

	case sinkport.SinkTypeNatsJetStream:
		var n NatsJetStreamSink
		if err := yaml.NodeToValue(value, &n, yaml.Strict()); err != nil {
			return nil, err
		}
		return &n, nil

	case sinkport.SinkTypeHTTP:
		var h HTTPSink
		if err := yaml.NodeToValue(value, &h, yaml.Strict()); err != nil {
			return nil, err
		}
		return &h, nil

	case sinkport.SinkTypeRabbitMQ:
		var r RabbitMQSink
		if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil {
			return nil, err
		}
		return &r, nil

	case sinkport.SinkTypeRabbitMQStream:
		var r RabbitMQStreamSink
		if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil {
			return nil, err
		}
		return &r, nil

	default:
		return nil, fmt.Errorf("line %d: sink: must define exactly one of %v", decoder.NodeLine(node), sinkport.SinkTypes)
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

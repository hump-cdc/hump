package sink

// Type is the YAML discriminator key that selects a concrete sink.
type Type string

// The sink types hump can deliver to; each literal is the key users write.
const (
	SinkTypeKafka          Type = "kafka"
	SinkTypeNatsCore       Type = "nats-core"
	SinkTypeNatsJetStream  Type = "nats-jetstream"
	SinkTypeHTTP           Type = "http"
	SinkTypeRabbitMQ       Type = "rabbitmq"
	SinkTypeRabbitMQStream Type = "rabbitmq-stream"
)

// SinkTypes lists every sink type; config parsing iterates it to find the
// discriminator, so a type missing here cannot be configured.
var SinkTypes = []Type{
	SinkTypeKafka,
	SinkTypeNatsCore,
	SinkTypeNatsJetStream,
	SinkTypeHTTP,
	SinkTypeRabbitMQ,
	SinkTypeRabbitMQStream,
}

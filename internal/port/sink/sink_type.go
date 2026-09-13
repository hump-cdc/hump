package sink

// SinkType is the YAML discriminator key that selects a concrete sink.

type SinkType string

const (
	SinkTypeKafka          SinkType = "kafka"
	SinkTypeNatsCore       SinkType = "nats-core"
	SinkTypeNatsJetStream  SinkType = "nats-jetstream"
	SinkTypeHTTP           SinkType = "http"
	SinkTypeRabbitMQ       SinkType = "rabbitmq"
	SinkTypeRabbitMQStream SinkType = "rabbitmq-stream"
)

var SinkTypes = []SinkType{
	SinkTypeKafka,
	SinkTypeNatsCore,
	SinkTypeNatsJetStream,
	SinkTypeHTTP,
	SinkTypeRabbitMQ,
	SinkTypeRabbitMQStream,
}

package sink

import (
	"hump/internal/types/wal"
	"time"
)

type SinkType string

const (
	SinkTypeKafka          SinkType = "kafka"
	SinkTypeNatsCore       SinkType = "nats-core"
	SinkTypeNatsJetStream  SinkType = "nats-jetstream"
	SinkTypeHTTP           SinkType = "http"
	SinkTypeRabbitMQ       SinkType = "rabbitmq"
	SinkTypeRabbitMQStream SinkType = "rabbitmq-stream"
)

var SinkTypes = []SinkType{SinkTypeKafka, SinkTypeNatsCore, SinkTypeNatsJetStream, SinkTypeHTTP, SinkTypeRabbitMQ, SinkTypeRabbitMQStream}

type Sink interface {
	Type() SinkType
	DisplayName() string
}

type SinkContext struct {
	Version string

	SinkType string
	SinkName string

	Timestamp time.Time
	Operation wal.WalOperation

	Database string
	Schema   string
	Table    string

	Headers map[string]string

	Prev map[string]any
	Next map[string]any
}

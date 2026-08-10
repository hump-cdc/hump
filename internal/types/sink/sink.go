package sink

import (
	"hump/internal/domain"
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
	Metadata(context SinkContext) (host string, target string)
}

type SinkContext struct {
	Version string

	SinkType string
	SinkName string

	Timestamp time.Time
	Operation domain.OperationType

	Database domain.DbName
	Schema   domain.TableSchema
	Table    domain.TableName

	Headers domain.Headers

	Prev domain.DbRecord
	Next domain.DbRecord
}

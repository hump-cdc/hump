package sink

import (
	"time"

	"hump/internal/domain"
)

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

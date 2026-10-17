package sink

import (
	"time"

	"hump/internal/domain"
)

// Context is one row change being delivered and the root of every sink
// template; its fields are what a template sees as .Operation, .Next and so on.
type Context struct {
	Version string

	Timestamp time.Time
	Operation domain.OperationType

	Database domain.DbName
	Schema   domain.TableSchema
	Table    domain.TableName

	Headers domain.Headers

	Prev domain.DbRecord
	Next domain.DbRecord
}

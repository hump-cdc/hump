package process_operation

import (
	"context"
	"hump/internal/domain"
	"time"
)

// Controller hands WAL row changes from the database port to the use case.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller delivering through useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Process delivers one row change; its signature matches database.ProcessMessageHandler.
func (c *Controller) Process(ctx context.Context, timestamp time.Time, operation domain.OperationType, dbName domain.DbName, schema domain.TableSchema, table domain.TableName, headers domain.Headers, prev domain.DbRecord, next domain.DbRecord) error {
	return c.useCase.Process(ctx, timestamp, operation, dbName, schema, table, headers, prev, next)
}

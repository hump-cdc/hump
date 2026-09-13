package process_operation

import (
	"context"
	"hump/internal/domain"
	"time"
)

type Controller struct {
	useCase *UseCase
}

func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

func (c *Controller) Process(ctx context.Context, timestamp time.Time, operation domain.OperationType, dbName domain.DbName, schema domain.TableSchema, table domain.TableName, headers domain.Headers, prev domain.DbRecord, next domain.DbRecord) error {
	return c.useCase.Process(ctx, timestamp, operation, dbName, schema, table, headers, prev, next)
}

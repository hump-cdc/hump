package wal

import (
	"context"
	"time"

	"hump/internal/domain"
)

type ProcessHandler func(
	ctx context.Context,
	timestamp time.Time,
	operation domain.OperationType,
	dbName domain.DbName,
	schema domain.TableSchema,
	table domain.TableName,
	headers domain.Headers,
	prev domain.DbRecord,
	next domain.DbRecord,
) (domain.DbName, domain.TableSchema, domain.TableName, error)

type Wal interface {
	Start(ctx context.Context) error
	Close(ctx context.Context) error
}

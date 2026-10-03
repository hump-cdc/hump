package database

import (
	"context"
	"hump/internal/domain"
	"time"
)

type Database interface {
	GetDbName() string
	GetDbHost() string
	GetDbPort() uint16
	GetPublicationName() string
	GetSlotName() string
	Listen(ctx context.Context, handler ProcessMessageHandler) error
	Cleanup(ctx context.Context) error
	GetTables(ctx context.Context, tables []domain.TableIdentifier) ([]TableDescription, error)
	GetRowCount(ctx context.Context, table domain.TableIdentifier, where string) (<-chan RowCount, error)
}

type ProcessMessageHandler func(
	ctx context.Context,

	timestamp time.Time,
	operation domain.OperationType,

	dbName domain.DbName,
	schema domain.TableSchema,
	table domain.TableName,

	headers domain.Headers,
	prev domain.DbRecord,
	next domain.DbRecord,
) error

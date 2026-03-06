package introspector

import (
	"context"
	"hump/internal/types/config"
)

type (
	TableName   = string
	TableSchema = string
)

type TableRef struct {
	Schema TableSchema
	Name   TableName
}

type Introspector interface {
	Invalidate(ctx context.Context, table *TableRef) error
	ResolveTableConfiguration(ctx context.Context, schemaName, name string) (TableRef, []config.SinkConfig, error)
}

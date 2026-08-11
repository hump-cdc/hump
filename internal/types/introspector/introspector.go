package introspector

import (
	"context"

	"hump/internal/domain"
	"hump/internal/port/sink"
)

type TableRef struct {
	Schema domain.TableSchema
	Name   domain.TableName
}

type Introspector interface {
	Invalidate(ctx context.Context, table *TableRef) error
	ResolveTableConfiguration(ctx context.Context, schemaName domain.TableSchema, name domain.TableName) (TableRef, []sink.Config, error)
}

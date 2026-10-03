// Package getsnapshottables serves the snapshot tables of the web UI: every
// configured table with its columns and a suggested snapshot filter.
package getsnapshottables

import (
	"context"

	"hump/internal/port/database"

	"github.com/danielgtaylor/huma/v2"
)

type Controller struct {
	useCase *UseCase
}

// NewController returns a controller serving the tables useCase describes.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}) (*SnapshotTablesResponse, error) {
	return func(ctx context.Context, _ *struct{}) (*SnapshotTablesResponse, error) {
		tables, err := c.useCase.Tables(ctx)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		resp := SnapshotTablesResponse{}
		resp.Body.Tables = make([]SnapshotTable, 0, len(tables))
		for _, table := range tables {
			resp.Body.Tables = append(resp.Body.Tables, SnapshotTable{
				Schema:  string(table.Table.Schema),
				Table:   string(table.Table.Name),
				Type:    string(table.Type),
				Columns: columns(table.Columns),
				Where:   table.Where,
				Sinks:   sinks(table.Sinks),
			})
		}

		return &resp, nil
	}
}

// columns renders the columns of a table.
func columns(source []database.TableColumn) []SnapshotColumn {
	result := make([]SnapshotColumn, 0, len(source))
	for _, c := range source {
		result = append(result, SnapshotColumn{Name: c.Name, Type: c.Type, Nullable: c.Nullable, PrimaryKey: c.PrimaryKey})
	}

	return result
}

// sinks renders the sinks of a table.
func sinks(source []Sink) []SnapshotSink {
	result := make([]SnapshotSink, 0, len(source))
	for _, s := range source {
		result = append(result, SnapshotSink{Type: s.Type, Host: s.Host, Target: s.Target})
	}

	return result
}

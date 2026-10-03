package getsnapshottables

import (
	"context"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
)

// fakeDatabase answers GetTables with scripted tables or an error, recording
// the tables it was asked for.
type fakeDatabase struct {
	database.Database

	tables []database.TableDescription
	err    error
	asked  []domain.TableIdentifier
}

// GetTables records tables and returns the scripted result.
func (d *fakeDatabase) GetTables(_ context.Context, tables []domain.TableIdentifier) ([]database.TableDescription, error) {
	d.asked = tables
	return d.tables, d.err
}

// configWith returns a configuration declaring the given tables.
func configWith(tables ...domain.TableIdentifier) config.Configuration {
	cfg := config.Configuration{}
	for _, t := range tables {
		cfg.Tables = append(cfg.Tables, config.Table{Schema: t.Schema, Name: t.Name})
	}

	return cfg
}

// identifier returns the table identifier of schema.name.
func identifier(schema, name string) domain.TableIdentifier {
	return domain.TableIdentifier{Schema: domain.TableSchema(schema), Name: domain.TableName(name)}
}

// fakeSink is a sink definition rendering a fixed host, and a target that is
// the table name when target is "{{table}}" and target itself otherwise.
type fakeSink struct {
	typ    sink.SinkType
	host   string
	target string
}

// Type returns the scripted sink type.
func (s fakeSink) Type() sink.SinkType { return s.typ }

// DisplayName returns the sink type.
func (s fakeSink) DisplayName() string { return string(s.typ) }

// Metadata returns the scripted host and the target rendered for ctx.
func (s fakeSink) Metadata(ctx sink.SinkContext) (string, string) {
	if s.target == "{{table}}" {
		return s.host, string(ctx.Table)
	}

	return s.host, s.target
}

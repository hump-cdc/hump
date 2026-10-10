package getsnapshottables

import (
	"context"
	"reflect"
	"testing"

	"hump/config"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
)

func TestTablesAsksForEveryConfiguredTableInOrder(t *testing.T) {
	db := &fakeDatabase{}
	cfg := configWith(identifier("public", "users"), identifier("audit", "events"))

	if _, err := NewUseCase(cfg, db).Tables(context.Background()); err != nil {
		t.Fatalf("Tables() error = %v", err)
	}

	want := []domain.TableIdentifier{identifier("public", "users"), identifier("audit", "events")}
	if !reflect.DeepEqual(db.asked, want) {
		t.Errorf("GetTables asked for %+v, want %+v", db.asked, want)
	}
}

func TestTablesListsDistinctSinkTargetsOfEachTable(t *testing.T) {
	db := &fakeDatabase{tables: []database.TableDescription{
		{Table: identifier("public", "users")},
		{Table: identifier("public", "posts")},
	}}

	kafka := fakeSink{typ: "kafka", host: "broker", target: "{{table}}"}
	http := fakeSink{typ: "http", host: "api", target: "/hook"}

	cfg := configWith(identifier("public", "users"), identifier("public", "posts"))
	cfg.Tables[0].Sinks = []sinkcfg.Config{
		{Sink: kafka, Operations: []domain.OperationType{domain.OperationTypeInsert}},
		{Sink: kafka, Operations: []domain.OperationType{domain.OperationTypeDelete}},
		{Sink: http},
		{},
	}

	got, err := NewUseCase(cfg, db).Tables(context.Background())
	if err != nil {
		t.Fatalf("Tables() error = %v", err)
	}

	want := [][]Sink{
		{{Type: "kafka", Host: "broker", Target: "users"}, {Type: "http", Host: "api", Target: "/hook"}},
		{},
	}
	for i, table := range got {
		if !reflect.DeepEqual(table.Sinks, want[i]) {
			t.Errorf("sinks of %s = %+v, want %+v", table.Table.Name, table.Sinks, want[i])
		}
	}
}

func TestTablesWithoutDatabaseIsEmpty(t *testing.T) {
	got, err := NewUseCase(configWith(identifier("public", "users")), nil).Tables(context.Background())
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("Tables() = %v, %v, want an empty list", got, err)
	}
}

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

package getsnapshottables

import (
	"context"
	"reflect"
	"testing"

	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/database"
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

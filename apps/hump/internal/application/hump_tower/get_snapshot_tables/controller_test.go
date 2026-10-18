package getsnapshottables

import (
	"context"
	"errors"
	"reflect"
	"testing"

	sinkcfg "hump/config/sink"
	"hump/internal/port/database"

	"github.com/danielgtaylor/huma/v2"
)

func TestHandlerRendersTables(t *testing.T) {
	db := &fakeDatabase{tables: []database.TableDescription{
		{
			Table: identifier("public", "events"),
			Type:  database.TableTypeHypertable,
			Columns: []database.TableColumn{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "time", Type: "timestamp with time zone", Nullable: true},
			},
			Where: `"time" >= now() - interval '7 days'`,
		},
		{Table: identifier("public", "users"), Type: database.TableTypeRegular},
	}}

	cfg := configWith(identifier("public", "events"))
	cfg.Tables[0].Sinks = []sinkcfg.Config{{Sink: fakeSink{typ: "kafka", host: "broker", target: "{{table}}"}}}

	resp, err := NewController(NewUseCase(cfg, db)).Handler()(context.Background(), nil)
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	want := []SnapshotTable{
		{
			Schema: "public",
			Table:  "events",
			Type:   "hypertable",
			Columns: []SnapshotColumn{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "time", Type: "timestamp with time zone", Nullable: true},
			},
			Where: `"time" >= now() - interval '7 days'`,
			Sinks: []SnapshotSink{{Type: "kafka", Host: "broker", Target: "events"}},
		},
		{Schema: "public", Table: "users", Type: "regular", Columns: []SnapshotColumn{}, Sinks: []SnapshotSink{}},
	}
	if !reflect.DeepEqual(resp.Body.Tables, want) {
		t.Fatalf("tables = %+v, want %+v", resp.Body.Tables, want)
	}
}

func TestHandlerRejectsADatabaseFailure(t *testing.T) {
	db := &fakeDatabase{err: errors.New("permission denied")}

	_, err := NewController(NewUseCase(configWith(identifier("public", "users")), db)).Handler()(context.Background(), nil)

	var statusErr huma.StatusError
	if !errors.As(err, &statusErr) || statusErr.GetStatus() != 422 {
		t.Fatalf("error = %v, want 422", err)
	}
}

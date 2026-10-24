package getconfig

import (
	"context"
	"reflect"
	"testing"

	"hump/config"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/pkg/app/options"
	"hump/pkg/version"
)

func TestHandlerRendersTheDescription(t *testing.T) {
	kafka := fakeSink{typ: "kafka", host: "broker", target: "events"}
	cfg := config.Configuration{Tables: []config.Table{{
		Schema:  "public",
		Name:    "users",
		Replica: domain.ReplicaIdentityFull,
		Sinks:   []sinkcfg.Config{{Sink: kafka, Operations: []domain.OperationType{domain.OperationTypeInsert}}},
	}}}
	opts := options.Options{ConfigPath: "/etc/hump.yaml", Config: "tables: []"}

	resp, err := NewController(NewUseCase(opts, cfg, fakeDatabase{})).Handler()(context.Background(), nil)
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	body := resp.Body
	if body.App.Version != version.Version || body.App.Commit != version.Commit {
		t.Errorf("App = %+v, want the build's version and commit", body.App)
	}
	if body.ConfigPath != opts.ConfigPath || body.RawConfig != opts.Config {
		t.Errorf("ConfigPath, RawConfig = %q, %q, want them from the options", body.ConfigPath, body.RawConfig)
	}

	db := body.Database
	if db.Host != "db.local" || db.Port != 5432 || db.Name != "app" || db.PublicationName != "hump_pub" || db.ReplicationSlotName != "hump_slot" {
		t.Errorf("Database = %+v, want the database's getters", db)
	}

	wantSinks := []ConfigSink{{Type: "kafka", Host: "broker"}}
	if !reflect.DeepEqual(body.Sinks, wantSinks) {
		t.Errorf("Sinks = %+v, want %+v", body.Sinks, wantSinks)
	}

	wantTables := []ConfigTable{{
		Schema:  "public",
		Name:    "users",
		Replica: "full",
		Sinks:   wantSinks,
		Targets: []ConfigTarget{{
			Type: "kafka", Host: "broker", Target: "events",
			Operations: []domain.OperationType{domain.OperationTypeInsert},
		}},
	}}
	if !reflect.DeepEqual(body.Tables, wantTables) {
		t.Errorf("Tables = %+v, want %+v", body.Tables, wantTables)
	}
}

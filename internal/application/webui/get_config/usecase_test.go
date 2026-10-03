package getconfig

import (
	"reflect"
	"testing"

	"hump/config"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
	"hump/pkg/app/options"
)

func TestConfigDescribesTheDatabaseAndOptions(t *testing.T) {
	opts := options.Options{ConfigPath: "/etc/hump.yaml", Config: "tables: []"}

	got := NewUseCase(opts, config.Configuration{}, fakeDatabase{}).Config()

	want := Database{Host: "db.local", Port: 5432, Name: "app", Publication: "hump_pub", Slot: "hump_slot"}
	if got.Database != want {
		t.Errorf("Database = %+v, want %+v", got.Database, want)
	}

	if got.ConfigPath != opts.ConfigPath || got.RawConfig != opts.Config {
		t.Errorf("ConfigPath, RawConfig = %q, %q, want them from the options", got.ConfigPath, got.RawConfig)
	}

	if got.Sinks == nil || got.Tables == nil {
		t.Errorf("Sinks, Tables = %v, %v, want empty lists rather than nil", got.Sinks, got.Tables)
	}
}

func TestConfigCollapsesSinksAndTargets(t *testing.T) {
	kafka := fakeSink{typ: "kafka", host: "broker", target: "events"}
	nats := fakeSink{typ: "nats", host: "nats", target: "{{table}}"}

	cfg := config.Configuration{Tables: []config.Table{
		{
			Schema:  "public",
			Name:    "users",
			Replica: domain.ReplicaIdentityFull,
			Sinks: []sinkcfg.Config{
				{Sink: kafka, Operations: []domain.OperationType{domain.OperationTypeDelete}},
				{Sink: kafka, Operations: []domain.OperationType{domain.OperationTypeInsert}},
				{Sink: nil},
				{Sink: nats, Operations: domain.OperationTypes},
			},
		},
		{
			Schema:  "public",
			Name:    "orders",
			Replica: domain.ReplicaIdentityUnmanaged,
			Sinks:   []sinkcfg.Config{{Sink: kafka, Operations: []domain.OperationType{domain.OperationTypeUpdate}}},
		},
	}}

	got := NewUseCase(options.Options{}, cfg, fakeDatabase{}).Config()

	wantSinks := []Sink{{Type: "kafka", Host: "broker"}, {Type: "nats", Host: "nats"}}
	if !reflect.DeepEqual(got.Sinks, wantSinks) {
		t.Errorf("Sinks = %+v, want each sink once: %+v", got.Sinks, wantSinks)
	}

	wantTables := []Table{
		{
			Schema:  "public",
			Name:    "users",
			Replica: "full",
			Sinks:   wantSinks,
			Targets: []Target{
				{Type: "kafka", Host: "broker", Target: "events", Operations: []domain.OperationType{domain.OperationTypeInsert, domain.OperationTypeDelete}},
				{Type: "nats", Host: "nats", Target: "users", Operations: domain.OperationTypes},
			},
		},
		{
			Schema:  "public",
			Name:    "orders",
			Replica: "unmanaged",
			Sinks:   []Sink{{Type: "kafka", Host: "broker"}},
			Targets: []Target{{Type: "kafka", Host: "broker", Target: "events", Operations: []domain.OperationType{domain.OperationTypeUpdate}}},
		},
	}
	if !reflect.DeepEqual(got.Tables, wantTables) {
		t.Errorf("Tables = %+v, want %+v", got.Tables, wantTables)
	}
}

// fakeDatabase answers the getters the description reads with fixed values.
type fakeDatabase struct {
	database.Database
}

func (fakeDatabase) GetDbHost() string          { return "db.local" }
func (fakeDatabase) GetDbPort() uint16          { return 5432 }
func (fakeDatabase) GetDbName() string          { return "app" }
func (fakeDatabase) GetPublicationName() string { return "hump_pub" }
func (fakeDatabase) GetSlotName() string        { return "hump_slot" }

// fakeSink is a sink definition rendering a fixed host, and a target that is
// the table name when target is "{{table}}" and target itself otherwise.
type fakeSink struct {
	typ    sink.SinkType
	host   string
	target string
}

func (s fakeSink) Type() sink.SinkType { return s.typ }
func (s fakeSink) DisplayName() string { return string(s.typ) }

func (s fakeSink) Metadata(ctx sink.SinkContext) (string, string) {
	if s.target == "{{table}}" {
		return s.host, string(ctx.Table)
	}

	return s.host, s.target
}

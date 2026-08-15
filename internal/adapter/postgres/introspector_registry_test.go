package postgres

import (
	"testing"

	"hump/config"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
)

func TestNewIntrospectorRegistryReplicaIdentityFullTables(t *testing.T) {
	cfg := config.Configuration{
		Tables: []config.Table{
			{Schema: "public", Name: "users", Replica: domain.ReplicaIdentityFull},
			{Schema: "public", Name: "events", Replica: domain.ReplicaIdentityUnmanaged},
			{Schema: "public", Name: "orders", Replica: domain.ReplicaIdentityFull},
		},
	}

	r := newIntrospectorRegistry(cfg)

	if got := len(r.List()); got != 3 {
		t.Fatalf("List(): want 3 tables, got %d", got)
	}

	full := r.ReplicaIdentityFullTables()
	want := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "orders"},
	}
	if len(full) != len(want) {
		t.Fatalf("ReplicaIdentityFullTables(): want %d, got %d: %+v", len(want), len(full), full)
	}
	for i, w := range want {
		if full[i] != w {
			t.Fatalf("ReplicaIdentityFullTables()[%d]: want %+v, got %+v", i, w, full[i])
		}
	}
}

func TestIntrospectorRegistryResolve(t *testing.T) {
	cfg := config.Configuration{
		Tables: []config.Table{
			{Schema: "public", Name: "events", Sinks: []sinkcfg.Config{{}}},
		},
	}

	r := newIntrospectorRegistry(cfg)
	r.UpdatePartitions(map[domain.TableIdentifier]domain.TableIdentifier{
		{Schema: "public", Name: "events_2024"}: {Schema: "public", Name: "events"},
	})

	cases := []struct {
		name  string
		table domain.TableIdentifier
		want  domain.TableIdentifier
		sinks int
	}{
		{
			name:  "configured table",
			table: domain.TableIdentifier{Schema: "public", Name: "events"},
			want:  domain.TableIdentifier{Schema: "public", Name: "events"},
			sinks: 1,
		},
		{
			name:  "partition of a configured table",
			table: domain.TableIdentifier{Schema: "public", Name: "events_2024"},
			want:  domain.TableIdentifier{Schema: "public", Name: "events"},
			sinks: 1,
		},
		{
			name:  "unknown table",
			table: domain.TableIdentifier{Schema: "other", Name: "logs"},
			want:  domain.TableIdentifier{},
			sinks: 0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, sinks := r.Resolve(tt.table)

			if got != tt.want {
				t.Errorf("Resolve(%+v) table = %+v, want %+v", tt.table, got, tt.want)
			}
			if len(sinks) != tt.sinks {
				t.Errorf("Resolve(%+v) sinks = %+v, want %d", tt.table, sinks, tt.sinks)
			}
		})
	}
}

func TestIntrospectorRegistryUpdatePartitionsCopiesInput(t *testing.T) {
	r := newIntrospectorRegistry(config.Configuration{})

	partition := domain.TableIdentifier{Schema: "public", Name: "events_2024"}
	input := map[domain.TableIdentifier]domain.TableIdentifier{
		partition: {Schema: "public", Name: "events"},
	}
	r.UpdatePartitions(input)

	delete(input, partition)
	input[domain.TableIdentifier{Schema: "public", Name: "events_2025"}] = domain.TableIdentifier{Schema: "public", Name: "events"}

	partitions := r.Partitions()
	if len(partitions) != 1 {
		t.Fatalf("Partitions() = %+v, want the one entry passed in", partitions)
	}
	if parent := partitions[partition]; parent.Name != "events" {
		t.Errorf("Partitions()[%+v] = %+v, want public.events", partition, parent)
	}
}

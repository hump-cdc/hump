package postgres

import (
	"testing"

	"hump/config"
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
			{Schema: "public", Name: "events"},
		},
	}

	r := newIntrospectorRegistry(cfg)
	r.UpdatePartitions(map[domain.TableIdentifier]domain.TableIdentifier{
		{Schema: "public", Name: "events_2024"}: {Schema: "public", Name: "events"},
		{Schema: "public", Name: "orphan_2024"}: {Schema: "public", Name: "orphan"},
		{Schema: "public", Name: "loop"}:        {Schema: "public", Name: "loop"},
	})

	cases := []struct {
		name  string
		table domain.TableIdentifier
		want  domain.TableIdentifier
		found bool
	}{
		{
			name:  "configured table",
			table: domain.TableIdentifier{Schema: "public", Name: "events"},
			want:  domain.TableIdentifier{Schema: "public", Name: "events"},
			found: true,
		},
		{
			name:  "partition of a configured table",
			table: domain.TableIdentifier{Schema: "public", Name: "events_2024"},
			want:  domain.TableIdentifier{Schema: "public", Name: "events"},
			found: true,
		},
		{
			name:  "partition whose parent is not configured",
			table: domain.TableIdentifier{Schema: "public", Name: "orphan_2024"},
			want:  domain.TableIdentifier{},
			found: false,
		},
		{
			// a self-referential entry used to recurse forever
			name:  "partition mapped to itself",
			table: domain.TableIdentifier{Schema: "public", Name: "loop"},
			want:  domain.TableIdentifier{},
			found: false,
		},
		{
			name:  "unknown table",
			table: domain.TableIdentifier{Schema: "other", Name: "logs"},
			want:  domain.TableIdentifier{},
			found: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, found := r.Resolve(tt.table)

			if got != tt.want {
				t.Errorf("Resolve(%+v) table = %+v, want %+v", tt.table, got, tt.want)
			}
			if found != tt.found {
				t.Errorf("Resolve(%+v) found = %v, want %v", tt.table, found, tt.found)
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

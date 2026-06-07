package introspection

import (
	"testing"

	"hump/internal/types/config"
	"hump/internal/types/introspector"
)

func TestRegistryFromConfig_ReplicaIdentityFullTables(t *testing.T) {
	cfg := config.Configuration{
		Tables: []config.Tables{
			{Schema: "public", Name: "users", Replica: config.ReplicaIdentityFull},
			{Schema: "public", Name: "events", Replica: config.ReplicaIdentityUnmanaged},
			{Schema: "public", Name: "orders", Replica: config.ReplicaIdentityFull},
		},
	}

	r := RegistryFromConfig(cfg)

	if got := len(r.List()); got != 3 {
		t.Fatalf("List(): want 3 tables, got %d", got)
	}

	full := r.ReplicaIdentityFullTables()
	want := []introspector.TableRef{
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

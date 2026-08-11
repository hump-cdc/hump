package introspection

import (
	"testing"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/types/introspector"
)

func TestRegistryFromConfig_ReplicaIdentityFullTables(t *testing.T) {
	cfg := config.Configuration{
		Tables: []config.Table{
			{Schema: "public", Name: "users", Replica: domain.ReplicaIdentityFull},
			{Schema: "public", Name: "events", Replica: domain.ReplicaIdentityUnmanaged},
			{Schema: "public", Name: "orders", Replica: domain.ReplicaIdentityFull},
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

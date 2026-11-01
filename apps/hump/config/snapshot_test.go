package config

import (
	"testing"

	"hump/internal/domain"
)

func TestResolveSnapshotTable(t *testing.T) {
	cases := []struct {
		name    string
		tables  []Table
		request domain.TableIdentifier
		want    domain.TableIdentifier
		ok      bool
	}{
		{"exact", []Table{{Schema: "private", Name: "users"}}, domain.TableIdentifier{Schema: "private", Name: "users"}, domain.TableIdentifier{Schema: "private", Name: "users"}, true},
		{"shorthand", []Table{{Schema: "private", Name: "users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{Schema: "private", Name: "users"}, true},
		{"other schema", []Table{{Schema: "private", Name: "users"}}, domain.TableIdentifier{Schema: "public", Name: "users"}, domain.TableIdentifier{}, false},
		{"unknown", []Table{{Schema: "public", Name: "users"}}, domain.TableIdentifier{Name: "orders"}, domain.TableIdentifier{}, false},
		{"duplicates", []Table{{Schema: "public", Name: "users"}, {Schema: "public", Name: "users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{Schema: "public", Name: "users"}, true},
		{"empty config schema", []Table{{Name: "users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{}, false},
		{"no public fallback", []Table{{Name: "users"}}, domain.TableIdentifier{Schema: "public", Name: "users"}, domain.TableIdentifier{}, false},
		{"empty schema after explicit", []Table{{Schema: "public", Name: "users"}, {Name: "users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{}, false},
		{"empty schema before explicit", []Table{{Name: "users"}, {Schema: "public", Name: "users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{}, false},
		{"exact despite ambiguity", []Table{{Name: "users"}, {Schema: "public", Name: "users"}, {Schema: "private", Name: "users"}}, domain.TableIdentifier{Schema: "public", Name: "users"}, domain.TableIdentifier{Schema: "public", Name: "users"}, true},
		{"ambiguous", []Table{{Schema: "public", Name: "users"}, {Schema: "private", Name: "users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{}, false},
		{"system catalog", []Table{{Schema: "public", Name: "users"}}, domain.TableIdentifier{Schema: "pg_catalog", Name: "pg_authid"}, domain.TableIdentifier{}, false},
		{"partition", []Table{{Schema: "public", Name: "users"}}, domain.TableIdentifier{Schema: "public", Name: "users_2026"}, domain.TableIdentifier{}, false},
		{"schema case sensitive", []Table{{Schema: "Public", Name: "users"}}, domain.TableIdentifier{Schema: "public", Name: "users"}, domain.TableIdentifier{}, false},
		{"name case sensitive", []Table{{Schema: "public", Name: "Users"}}, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{}, false},
		{"empty configuration", nil, domain.TableIdentifier{Name: "users"}, domain.TableIdentifier{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Configuration{Tables: tc.tables}
			got, ok := cfg.ResolveSnapshotTable(tc.request)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("ResolveSnapshotTable = %+v, %v; want %+v, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

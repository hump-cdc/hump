package postgres

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestNewBootstrapperBorrowsConnection(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)

	if bootstrapper.connection == nil {
		t.Fatal("newBootstrapper() left the connection nil")
	}
}

func TestEnsurePublicationExists(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(true,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
		layoutRow("public", "metrics", tableTypeHypertable, []byte("_timescaledb_internal"), []byte("_hyper_1_1")),
	))
	bootstrapper := newTestBootstrapper(t, server)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "metrics"},
	}

	if err := bootstrapper.EnsurePublicationExists(context.Background(), "hump_pub", tables); err != nil {
		t.Fatalf("EnsurePublicationExists() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 3 {
		t.Fatalf("statements = %d, want 3: %q", len(statements), statements)
	}

	publication := statements[2]
	want := `TABLE "public"."metrics", "public"."users", TABLES IN SCHEMA "_timescaledb_internal"`
	if !strings.Contains(publication, `ALTER PUBLICATION "hump_pub" SET `+want) {
		t.Errorf("publication statement = %q, want the ALTER to set %q", publication, want)
	}
	if !strings.Contains(publication, `CREATE PUBLICATION "hump_pub" FOR `+want) {
		t.Errorf("publication statement = %q, want the CREATE to declare %q", publication, want)
	}
	if !strings.Contains(publication, `pubname = 'hump_pub'`) {
		t.Errorf("publication statement = %q, want the name as a literal in the existence check", publication)
	}
}

func TestEnsurePublicationExistsSkipsUnnamedChunkSchema(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(true,
		layoutRow("public", "metrics", tableTypeHypertable, []byte(""), []byte("_hyper_1_1")),
	))
	bootstrapper := newTestBootstrapper(t, server)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "metrics"}}

	if err := bootstrapper.EnsurePublicationExists(context.Background(), "hump_pub", tables); err != nil {
		t.Fatalf("EnsurePublicationExists() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 3 {
		t.Fatalf("statements = %d, want 3: %q", len(statements), statements)
	}
	if strings.Contains(statements[2], "TABLES IN SCHEMA") {
		t.Errorf("publication statement = %q, want no schema clause for a chunk schema with no name", statements[2])
	}
	if !strings.Contains(statements[2], `TABLE "public"."metrics"`) {
		t.Errorf("publication statement = %q, want the hypertable itself still listed", statements[2])
	}
}

func TestEnsurePublicationExistsPropagatesLookupError(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsurePublicationExists(context.Background(), "hump_pub", nil)
	if !errors.Is(err, database.ErrNoTablesConfigured) {
		t.Fatalf("EnsurePublicationExists() error = %v, want database.ErrNoTablesConfigured", err)
	}
}

func TestEnsurePublicationExistsWrapsServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "pg_extension"):
			return singleValue("f")
		case strings.Contains(sql, "unnest("):
			return tableLayout(layoutRow("public", "users", tableTypeRegular, nil, nil))
		default:
			return serverError("42501", "permission denied")
		}
	})
	bootstrapper := newTestBootstrapper(t, server)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	err := bootstrapper.EnsurePublicationExists(context.Background(), "hump_pub", tables)
	if err == nil {
		t.Fatal("EnsurePublicationExists() error = nil, want the server error")
	}
	if !strings.Contains(err.Error(), `ensure publication "hump_pub"`) {
		t.Errorf("error = %v, want it to name the publication", err)
	}
}

func TestWarnMissingReplicaIdentityQueriesThePublication(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult {
		return publishedTables(domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_materialized_hypertable_2"})
	})
	bootstrapper := newTestBootstrapper(t, server)

	bootstrapper.WarnMissingReplicaIdentity(context.Background(), "hump_pub")

	statements := server.recorded()
	if len(statements) != 1 {
		t.Fatalf("statements = %q, want the preflight query", statements)
	}
	if !strings.Contains(statements[0], "pg_publication_tables") {
		t.Errorf("statement = %q, want the preflight query", statements[0])
	}

	params := server.boundParams()
	if len(params) != 1 {
		t.Fatalf("bound parameter sets = %d, want 1", len(params))
	}
	if got, want := string(params[0][0]), "hump_pub"; got != want {
		t.Errorf("publication = %q, want %q", got, want)
	}
}

func TestWarnMissingReplicaIdentitySwallowsQueryError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	bootstrapper := newTestBootstrapper(t, server)

	bootstrapper.WarnMissingReplicaIdentity(context.Background(), "hump_pub")

	if statements := server.recorded(); len(statements) != 1 {
		t.Fatalf("statements = %q, want the preflight query to have been attempted", statements)
	}
}

func TestGetPublishedTablesMissingReplicaIdentity(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult {
		return publishedTables(
			domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_9_9_chunk"},
			domain.TableIdentifier{Schema: "public", Name: "logs"},
		)
	})
	bootstrapper := newTestBootstrapper(t, server)

	tables, err := bootstrapper.getPublishedTablesMissingReplicaIdentity(context.Background(), "hump_pub")
	if err != nil {
		t.Fatalf("getPublishedTablesMissingReplicaIdentity() error = %v", err)
	}

	want := []domain.TableIdentifier{
		{Schema: "_timescaledb_internal", Name: "_hyper_9_9_chunk"},
		{Schema: "public", Name: "logs"},
	}
	if !slices.Equal(tables, want) {
		t.Errorf("tables = %+v, want %+v", tables, want)
	}
}

func TestGetPublishedTablesMissingReplicaIdentityRejectsShortRows(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult {
		return queryResult{columns: 1, rows: [][][]byte{{[]byte("public")}}}
	})
	bootstrapper := newTestBootstrapper(t, server)

	_, err := bootstrapper.getPublishedTablesMissingReplicaIdentity(context.Background(), "hump_pub")
	if err == nil {
		t.Fatal("getPublishedTablesMissingReplicaIdentity() error = nil, want the column count error")
	}
	if !strings.Contains(err.Error(), "expected 2 columns") {
		t.Errorf("error = %v, want it to report the column count", err)
	}
}

func TestGetPublishedTablesMissingReplicaIdentityWrapsQueryError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	bootstrapper := newTestBootstrapper(t, server)

	_, err := bootstrapper.getPublishedTablesMissingReplicaIdentity(context.Background(), "hump_pub")
	if err == nil {
		t.Fatal("getPublishedTablesMissingReplicaIdentity() error = nil, want the server error")
	}
	if !strings.Contains(err.Error(), "get published tables missing replica identity") {
		t.Errorf("error = %v, want it wrapped", err)
	}
}

func TestEnsurePublicationRemoved(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)

	if err := bootstrapper.EnsurePublicationRemoved(context.Background(), `we"ird`); err != nil {
		t.Fatalf("EnsurePublicationRemoved() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 1 {
		t.Fatalf("statements = %d, want 1: %q", len(statements), statements)
	}
	if !strings.Contains(statements[0], `DROP PUBLICATION IF EXISTS "we""ird"`) {
		t.Errorf("statement = %q, want the publication name quoted as an identifier", statements[0])
	}
}

func TestEnsurePublicationRemovedWrapsServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsurePublicationRemoved(context.Background(), "hump_pub")
	if err == nil {
		t.Fatal("EnsurePublicationRemoved() error = nil, want the server error")
	}
	if !strings.Contains(err.Error(), `remove publication "hump_pub"`) {
		t.Errorf("error = %v, want it to name the publication", err)
	}
}

func TestEnsureReplicationSlotExists(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)

	if err := bootstrapper.EnsureReplicationSlotExists(context.Background(), "hump's slot"); err != nil {
		t.Fatalf("EnsureReplicationSlotExists() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 1 {
		t.Fatalf("statements = %d, want 1: %q", len(statements), statements)
	}
	if !strings.Contains(statements[0], "pg_create_logical_replication_slot") {
		t.Errorf("statement = %q, want the slot creation block", statements[0])
	}
	if !strings.Contains(statements[0], `'hump''s slot'`) {
		t.Errorf("statement = %q, want the slot name quoted as a literal", statements[0])
	}
}

func TestEnsureReplicationSlotExistsWrapsServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("55006", "slot is active") })
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsureReplicationSlotExists(context.Background(), "hump_slot")
	if err == nil {
		t.Fatal("EnsureReplicationSlotExists() error = nil, want the server error")
	}

	if !strings.Contains(err.Error(), `ensure replication slot "hump_slot"`) {
		t.Errorf("error = %v, want it to name the slot", err)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55006" {
		t.Errorf("error = %v, want a wrapped *pgconn.PgError with SQLSTATE 55006", err)
	}
}

func TestEnsureReplicationSlotRemoved(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)

	if err := bootstrapper.EnsureReplicationSlotRemoved(context.Background(), "hump_slot"); err != nil {
		t.Fatalf("EnsureReplicationSlotRemoved() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 1 {
		t.Fatalf("statements = %d, want 1: %q", len(statements), statements)
	}
	if !strings.Contains(statements[0], "pg_drop_replication_slot") {
		t.Errorf("statement = %q, want the slot removal block", statements[0])
	}
	if !strings.Contains(statements[0], `'hump_slot'`) {
		t.Errorf("statement = %q, want the slot name quoted as a literal", statements[0])
	}
}

func TestEnsureReplicationSlotRemovedWrapsServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("55006", "slot is active") })
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsureReplicationSlotRemoved(context.Background(), "hump_slot")
	if err == nil {
		t.Fatal("EnsureReplicationSlotRemoved() error = nil, want the server error")
	}

	if !strings.Contains(err.Error(), `remove replication slot "hump_slot"`) {
		t.Errorf("error = %v, want it to name the slot", err)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55006" {
		t.Errorf("error = %v, want a wrapped *pgconn.PgError with SQLSTATE 55006", err)
	}
}

func TestEnsureReplicaIdentityFullAltersOnlyStaleTables(t *testing.T) {
	users := domain.TableIdentifier{Schema: "public", Name: "users"}
	events := domain.TableIdentifier{Schema: "public", Name: "events"}

	cases := []struct {
		name  string
		stale []domain.TableIdentifier
		want  []string
	}{
		{name: "every table already full", stale: nil, want: nil},
		{name: "one table behind", stale: []domain.TableIdentifier{users}, want: []string{`"public"."users"`}},
		{
			name:  "several tables behind",
			stale: []domain.TableIdentifier{users, events},
			want:  []string{`"public"."users"`, `"public"."events"`},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(func(string) queryResult { return staleReplicaIdentity(tt.stale...) })
			bootstrapper := newTestBootstrapper(t, server)

			if err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), []domain.TableIdentifier{users, events}); err != nil {
				t.Fatalf("EnsureReplicaIdentityFull() error = %v", err)
			}

			got := alteredTables(server)
			if len(got) != len(tt.want) {
				t.Fatalf("altered tables = %q, want %q", got, tt.want)
			}
			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("altered table %d = %q, want %q", i, got[i], want)
				}
			}

			if statements := server.recorded(); len(statements) != len(tt.want)+1 {
				t.Errorf("statements = %d, want %d: one check plus one ALTER per stale table", len(statements), len(tt.want)+1)
			}
		})
	}
}

func TestEnsureReplicaIdentityFullBindsTablesAsArrays(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return staleReplicaIdentity() })
	bootstrapper := newTestBootstrapper(t, server)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "other", Name: "events"},
	}

	if err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), tables); err != nil {
		t.Fatalf("EnsureReplicaIdentityFull() error = %v", err)
	}

	params := server.boundParams()
	if len(params) != 1 {
		t.Fatalf("bound parameter sets = %d, want 1", len(params))
	}
	if got, want := string(params[0][0]), "{public,other}"; got != want {
		t.Errorf("schemas = %q, want %q", got, want)
	}
	if got, want := string(params[0][1]), "{users,events}"; got != want {
		t.Errorf("names = %q, want %q", got, want)
	}
}

func TestEnsureReplicaIdentityFullQuotesTheTable(t *testing.T) {
	table := domain.TableIdentifier{Schema: "pub'lic", Name: "us\"ers"}

	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return staleReplicaIdentity(table) })
	bootstrapper := newTestBootstrapper(t, server)

	if err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), []domain.TableIdentifier{table}); err != nil {
		t.Fatalf("EnsureReplicaIdentityFull() error = %v", err)
	}

	statements := server.recorded()
	if len(statements) != 2 {
		t.Fatalf("statements = %d, want 2: %q", len(statements), statements)
	}

	if alter := statements[1]; !strings.Contains(alter, `ALTER TABLE "pub'lic"."us""ers" REPLICA IDENTITY FULL`) {
		t.Errorf("statement = %q, want the table quoted as an identifier", alter)
	}
}

func TestEnsureReplicaIdentityFullSkipsAnEmptyTableList(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)

	if err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), nil); err != nil {
		t.Fatalf("EnsureReplicaIdentityFull() error = %v", err)
	}

	if statements := server.recorded(); len(statements) != 0 {
		t.Errorf("statements = %q, want none: an empty list must not reach the server", statements)
	}
}

func TestEnsureReplicaIdentityFullPropagatesCheckError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), []domain.TableIdentifier{{Schema: "public", Name: "users"}})
	if err == nil {
		t.Fatal("EnsureReplicaIdentityFull() error = nil, want the server error")
	}
	if !strings.Contains(err.Error(), "get stale replica identity tables") {
		t.Errorf("error = %v, want it to name the check", err)
	}
}

func TestEnsureReplicaIdentityFullWrapsServerError(t *testing.T) {
	users := domain.TableIdentifier{Schema: "public", Name: "users"}

	server := newFakeServer(t)
	server.setResponder(func(sql string) queryResult {
		if strings.Contains(sql, "relreplident") {
			return staleReplicaIdentity(users)
		}

		return serverError("42501", "permission denied")
	})
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), []domain.TableIdentifier{users})
	if err == nil {
		t.Fatal("EnsureReplicaIdentityFull() error = nil, want the server error")
	}
	if !strings.Contains(err.Error(), "set replica identity full on public.users") {
		t.Errorf("error = %v, want it to name the table", err)
	}
}

func TestEnsureReplicaIdentityFullWrapsEncodeError(t *testing.T) {
	server := newFakeServer(t)
	bootstrapper := newTestBootstrapper(t, server)
	bootstrapper.newTypeMap = newUnencodableTypeMap

	err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), []domain.TableIdentifier{{Schema: "public", Name: "users"}})
	if err == nil {
		t.Fatal("EnsureReplicaIdentityFull() error = nil, want the encode error")
	}
	if !strings.Contains(err.Error(), "encode schemas array") {
		t.Errorf("error = %v, want it to name the array that failed", err)
	}

	if statements := server.recorded(); len(statements) != 0 {
		t.Errorf("statements = %q, want none: the check must not run", statements)
	}
}

func TestEnsureReplicaIdentityFullRejectsShortRows(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult {
		return queryResult{columns: 1, rows: [][][]byte{{[]byte("public")}}}
	})
	bootstrapper := newTestBootstrapper(t, server)

	err := bootstrapper.EnsureReplicaIdentityFull(context.Background(), []domain.TableIdentifier{{Schema: "public", Name: "users"}})
	if err == nil {
		t.Fatal("EnsureReplicaIdentityFull() error = nil, want the column count error")
	}
	if !strings.Contains(err.Error(), "expected 2 columns, got 1") {
		t.Errorf("error = %v, want it to report the column count", err)
	}
}

func TestQuoteLiteral(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "hump_slot", want: `'hump_slot'`},
		{name: "empty", in: "", want: `''`},
		{name: "embedded quote", in: "o'brien", want: `'o''brien'`},
		{name: "only quotes", in: `''`, want: `''''''`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			b := &bootstrapper{}

			if got := b.quoteLiteral(tt.in); got != tt.want {
				t.Errorf("quoteLiteral(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "users", want: `"users"`},
		{name: "empty", in: "", want: `""`},
		{name: "embedded quote", in: `we"ird`, want: `"we""ird"`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			b := &bootstrapper{}

			if got := b.quoteIdentifier(tt.in); got != tt.want {
				t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// alteredTables returns the qualified table of every ALTER TABLE ... REPLICA
// IDENTITY FULL the server was asked to run, in order.
func TestSummarizeTables(t *testing.T) {
	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "a"},
		{Schema: "public", Name: "b"},
		{Schema: "public", Name: "c"},
	}

	cases := []struct {
		name   string
		tables []domain.TableIdentifier
		limit  int
		want   string
	}{
		{name: "none", tables: nil, limit: 2, want: ""},
		{name: "under the limit", tables: tables[:2], limit: 2, want: "public.a, public.b"},
		{name: "over the limit", tables: tables, limit: 2, want: "public.a, public.b, ... and 1 more"},
	}

	bootstrapper := newTestBootstrapper(t, newFakeServer(t))

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := bootstrapper.summarizeTables(tt.tables, tt.limit); got != tt.want {
				t.Errorf("summarizeTables() = %q, want %q", got, tt.want)
			}
		})
	}
}

func alteredTables(s *fakeServer) []string {
	alterPattern := regexp.MustCompile(`ALTER TABLE (.+) REPLICA IDENTITY FULL`)

	var altered []string
	for _, statement := range s.recorded() {
		if match := alterPattern.FindStringSubmatch(statement); match != nil {
			altered = append(altered, match[1])
		}
	}

	return altered
}

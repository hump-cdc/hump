//go:build integration

package postgres

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestPostgresStringLiteralsIntegrationRoundTrip(t *testing.T) {
	_, conn := integrationStringConnection(t)
	ctx := t.Context()

	for _, setting := range []string{"on", "off"} {
		t.Run(setting, func(t *testing.T) {
			integrationStringExec(ctx, t, conn, "SET standard_conforming_strings = "+setting)
			raw, err := conn.Raw(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got := raw.ParameterStatus("standard_conforming_strings"); got != setting {
				t.Fatalf("effective setting = %q, want %q", got, setting)
			}
			for i, value := range integrationStringValues() {
				results, err := conn.Exec(ctx, "SELECT "+quoteLiteral(value))
				if err != nil {
					t.Fatalf("value %d: %v", i, err)
				}
				if len(results) != 1 || len(results[0].Rows) != 1 || len(results[0].Rows[0]) != 1 {
					t.Fatalf("value %d: expected exactly one result, row and column", i)
				}
				if !bytes.Equal(results[0].Rows[0][0], []byte(value)) {
					t.Errorf("value %d: round-trip = %q, want %q", i, results[0].Rows[0][0], value)
				}
			}
		})
	}
}

func TestPostgresStringLiteralsIntegrationDefaultAndReconnect(t *testing.T) {
	dsn, _ := integrationStringConnection(t)
	ctx := t.Context()
	overrideDSN := integrationStringOverride(t, dsn, "on")
	cfg, err := defaultSettingsConfig(overrideDSN)
	if err != nil {
		t.Fatal("parse diagnostic configuration")
	}
	diagnostic, err := pgconn.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal("connect diagnostic session")
	}
	t.Cleanup(func() { closeDetached(diagnostic.Close) })
	results, err := diagnostic.Exec(ctx, standardConformingStringsSQL).ReadAll()
	if err != nil {
		t.Fatal("read effective default")
	}
	value, err := rowCountValue(results)
	if err != nil || string(value) != "off" {
		t.Fatalf("effective role/database default = %q (error %v); use an isolated server started with standard_conforming_strings=off", value, err)
	}

	var logs bytes.Buffer
	originalLogger := logger.Logger
	logger.Logger = slog.New(slog.NewJSONHandler(&logs, nil))
	t.Cleanup(func() { logger.Logger = originalLogger })
	checkStandardConformingStringsDefault(ctx, overrideDSN)
	if !strings.Contains(logs.String(), `"level":"WARN"`) {
		t.Fatalf("default off did not produce WARN: %s", logs.String())
	}

	for _, replication := range []bool{false, true} {
		t.Run(fmt.Sprintf("replication=%t", replication), func(t *testing.T) {
			opts := []option{withMaxAttempts(1)}
			if replication {
				opts = append(opts, withReplication())
			}
			conn, err := newConnection(integrationStringOverride(t, dsn, "off"), opts...)
			if err != nil {
				t.Fatal("create working connection")
			}
			t.Cleanup(func() { closeDetached(conn.Close) })
			var previous *pgconn.PgConn
			for attempt := range 2 {
				raw, err := conn.Raw(ctx)
				if err != nil {
					t.Fatalf("acquire %d: %v", attempt, err)
				}
				if raw == previous {
					t.Fatal("closed session was reused")
				}
				if got := raw.ParameterStatus("standard_conforming_strings"); got != "on" {
					t.Fatalf("acquire %d: effective setting = %q, want on", attempt, got)
				}
				integrationStringExec(ctx, t, conn, "SELECT 1")
				previous = raw
				if err := raw.Close(ctx); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPostgresStringLiteralsIntegrationFiltersCountSnapshotAndPublication(t *testing.T) {
	dsn, conn := integrationStringPlainDatabase(t)
	ctx := t.Context()
	name := fmt.Sprintf("hump_literals_%d", time.Now().UnixNano())
	table := domain.TableIdentifier{Schema: "public", Name: domain.TableName(name)}
	qualified := qualifiedTableName(table)
	integrationStringExec(ctx, t, conn, "CREATE TABLE "+qualified+" (id integer PRIMARY KEY, value text NOT NULL)")
	t.Cleanup(func() { integrationStringExec(context.WithoutCancel(ctx), t, conn, "DROP TABLE IF EXISTS "+qualified) })
	values := integrationStringValues()
	for i, value := range values {
		if _, err := conn.ExecParams(ctx, "INSERT INTO "+qualified+" VALUES ($1::integer, $2::text)", []param{textParam(strconv.Itoa(i)), textParam(value)}, nil); err != nil {
			t.Fatalf("insert value %d: %v", i, err)
		}
	}

	publication := name + `_'\"`
	bootstrap := newBootstrapper(conn)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if err := bootstrap.EnsurePublicationRemoved(cleanupCtx, publication); err != nil {
			t.Errorf("remove publication: %v", err)
		}
	})
	for _, setting := range []string{"on", "off"} {
		integrationStringExec(ctx, t, conn, "SET standard_conforming_strings = "+setting)
		for range 2 {
			if err := bootstrap.EnsurePublicationExists(ctx, publication, []domain.TableIdentifier{table}); err != nil {
				t.Fatalf("ensure publication with strings %s: %v", setting, err)
			}
		}
		result, err := conn.ExecParams(ctx, "SELECT count(*) FROM pg_publication WHERE pubname = $1", []param{textParam(publication)}, nil)
		if err != nil || len(result.Rows) != 1 || string(result.Rows[0][0]) != "1" {
			t.Fatalf("publication lookup with strings %s: result %v, error %v", setting, result, err)
		}
		for i, value := range values {
			where := integrationStringWhere(value)
			condition, err := newSnapshotWhereParser().Parse(where)
			if err != nil {
				t.Fatalf("parse value %d: %v", i, err)
			}
			results, err := conn.Exec(ctx, "SELECT id FROM "+qualified+" WHERE "+condition)
			if err != nil {
				t.Fatalf("filter value %d with strings %s: %v", i, setting, err)
			}
			if len(results) != 1 || len(results[0].Rows) != 1 || string(results[0].Rows[0][0]) != strconv.Itoa(i) {
				t.Fatalf("filter value %d with strings %s returned unexpected rows: %v", i, setting, results)
			}
		}
	}

	p := &Postgres{databaseUrl: dsn}
	snapshot := newSnapshot(dsn, newWalGate(), newMetrics("integration", promauto.With(prometheus.NewRegistry())))
	id, err := snapshot.Start(ctx, []domain.TableIdentifier{table})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if err := snapshot.Finish(cleanupCtx); err != nil {
			t.Errorf("finish snapshot: %v", err)
		}
	})
	for i, value := range values {
		where := integrationStringWhere(value)
		counts, err := p.GetRowCount(ctx, table, where)
		if err != nil {
			t.Fatalf("count value %d: %v", i, err)
		}
		exact := false
		for count := range counts {
			if count.Error != nil {
				t.Fatal(count.Error)
			}
			if !count.Rough {
				exact = true
				if count.Num != 1 {
					t.Fatalf("exact count value %d = %d, want 1", i, count.Num)
				}
			}
		}
		if !exact {
			t.Fatalf("no exact count for value %d", i)
		}
		var records []domain.DbRecord
		if err := snapshot.Read(ctx, id, table, where, func(record domain.DbRecord, metadata database.SnapshotRowMetadata) error {
			records = append(records, record)
			if metadata.Table != table.Name || metadata.Schema != table.Schema || metadata.CTID == "" {
				t.Errorf("unexpected snapshot metadata: %+v", metadata)
			}
			return nil
		}); err != nil {
			t.Fatalf("snapshot value %d: %v", i, err)
		}
		if len(records) != 1 || records[0]["value"] != value {
			t.Fatalf("snapshot value %d = %#v, want one row containing %q", i, records, value)
		}
	}
}

func TestPostgresStringLiteralsIntegrationDatabaseDSN(t *testing.T) {
	for _, dsn := range []string{
		"postgres://user:secret@localhost/original?dbname=override&application_name=keep%20spaces",
		"host=localhost dbname=original user=user password=secret application_name='keep spaces'",
	} {
		isolated := integrationStringDatabaseDSN(t, dsn, "hump_literals_test")
		cfg, err := pgconn.ParseConfig(isolated)
		if err != nil {
			t.Fatal("parse isolated database DSN")
		}
		if cfg.Database != "hump_literals_test" || cfg.User != "user" || cfg.Password != "secret" || cfg.RuntimeParams["application_name"] != "keep spaces" {
			t.Fatal("isolated DSN did not preserve connection settings and replace database")
		}
	}
}

// integrationStringPlainDatabase avoids the unrelated TimescaleDB catalog path.
// A preselected plain database needs no CREATEDB privilege; otherwise only the new database is removed.
func integrationStringPlainDatabase(t *testing.T) (string, *connection) {
	t.Helper()
	dsn, admin := integrationStringConnection(t)
	timescale, err := newTableMetadataLoader(admin).hasTimescaleDB(t.Context())
	if err != nil {
		t.Fatal("probe integration database extensions")
	}
	if !timescale {
		return dsn, admin
	}

	name := fmt.Sprintf("hump_literals_db_%d_%d", os.Getpid(), time.Now().UnixNano())
	if _, err := admin.Exec(t.Context(), "CREATE DATABASE "+quoteIdentifier(name)+" TEMPLATE template0"); err != nil {
		t.Fatalf("create isolated plain database (requires CREATEDB, not superuser; alternatively select a plain database in HUMP_TEST_POSTGRES_DSN): %v", err)
	}
	t.Cleanup(func() {
		integrationStringExec(context.WithoutCancel(t.Context()), t, admin, "DROP DATABASE "+quoteIdentifier(name))
	})

	dsn = integrationStringDatabaseDSN(t, dsn, name)
	conn, err := newConnection(dsn, withMaxAttempts(1))
	if err != nil {
		t.Fatal("parse isolated database connection")
	}
	t.Cleanup(func() { closeDetached(conn.Close) })
	timescale, err = newTableMetadataLoader(conn).hasTimescaleDB(t.Context())
	if err != nil || timescale {
		t.Fatalf("isolated template0 database must not contain timescaledb: installed=%t, error=%v", timescale, err)
	}
	return dsn, conn
}

// integrationStringDatabaseDSN changes only the database, retaining both supported DSN formats.
func integrationStringDatabaseDSN(t *testing.T, dsn string, name string) string {
	t.Helper()
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatal("parse integration DSN URL")
		}
		parsed.Path = "/" + name
		parsed.RawPath = ""
		query := parsed.Query()
		query.Del("dbname")
		query.Del("database")
		parsed.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
		return parsed.String()
	}
	return dsn + " dbname='" + strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(name) + "'"
}

// integrationStringConnection opens only an explicitly selected PostgreSQL 17 instance.
func integrationStringConnection(t *testing.T) (string, *connection) {
	t.Helper()
	dsn := os.Getenv("HUMP_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set HUMP_TEST_POSTGRES_DSN to an isolated PostgreSQL 17 instance with standard_conforming_strings=off")
	}
	conn, err := newConnection(dsn, withMaxAttempts(1))
	if err != nil {
		t.Fatal("parse HUMP_TEST_POSTGRES_DSN")
	}
	t.Cleanup(func() { closeDetached(conn.Close) })
	raw, err := conn.Raw(t.Context())
	if err != nil {
		t.Fatal("connect to HUMP_TEST_POSTGRES_DSN")
	}
	version := raw.ParameterStatus("server_version")
	t.Logf("PostgreSQL server_version=%s", version)
	if !strings.HasPrefix(version, "17.") {
		t.Fatalf("server_version = %q, want PostgreSQL 17", version)
	}
	return dsn, conn
}

// integrationStringExec executes fixture SQL, retaining a live context during cleanup.
func integrationStringExec(ctx context.Context, t *testing.T, conn *connection, statement string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, closeTimeout)
	defer cancel()
	if _, err := conn.Exec(ctx, statement); err != nil {
		t.Fatal(err)
	}
}

// integrationStringOverride adds conflicting client overrides without changing the server.
func integrationStringOverride(t *testing.T, dsn string, optionSetting string) string {
	t.Helper()
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatal("parse integration DSN URL")
		}
		query := parsed.Query()
		query.Set("standard_conforming_strings", "off")
		query.Set("options", "-c standard_conforming_strings="+optionSetting)
		parsed.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
		return parsed.String()
	}
	return dsn + " standard_conforming_strings=off options='-c standard_conforming_strings=" + optionSetting + "'"
}

// integrationStringWhere uses the input grammar, not the SQL renderer's E-literal syntax.
func integrationStringWhere(value string) string {
	return "value = '" + strings.ReplaceAll(value, "'", "''") + "'"
}

// integrationStringValues covers byte preservation and SQL-looking text without NUL bytes.
func integrationStringValues() []string {
	return []string{"", "O'Reilly", `back\slash`, `trailing\`, "line\nfeed\r\nnext", `\'; SELECT 999; --`, `'); DROP TABLE hump_literals_guard; --`, `quote'\backslash`, "кириллица 🐘"}
}

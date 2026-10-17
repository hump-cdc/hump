package postgres

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestGetTablesDescribesTablesInRequestedOrder(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tablesResponder("t", tableColumns(
		columnRow("public", "users", "regular", "id", "bigint", "f", "t", "f", "f", "t"),
		columnRow("public", "users", "regular", "updated_at", "timestamp with time zone", "t", "f", "f", "t", "f"),
		columnRow("public", "events", "hypertable", "time", "timestamp with time zone", "f", "f", "t", "t", "t"),
		columnRow("public", "events", "hypertable", "kind", "text", "f", "f", "f", "f", "f"),
	)))

	p, _, _ := newTestPostgres(t, server)

	got, err := p.GetTables(context.Background(), []domain.TableIdentifier{
		eventsTable(),
		{Schema: "public", Name: "missing"},
		usersTable(),
	})
	if err != nil {
		t.Fatalf("GetTables() error = %v", err)
	}

	want := []database.TableDescription{
		{
			Table: eventsTable(),
			Type:  database.TableTypeHypertable,
			Columns: []database.TableColumn{
				{Name: "time", Type: "timestamp with time zone"},
				{Name: "kind", Type: "text"},
			},
			Where: `"time" >= now() - interval '7 days'`,
		},
		{
			Table: usersTable(),
			Type:  database.TableTypeRegular,
			Columns: []database.TableColumn{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "updated_at", Type: "timestamp with time zone", Nullable: true},
			},
			Where: `"updated_at" >= now() - interval '7 days'`,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetTables() = %+v, want %+v", got, want)
	}

	params := server.boundParams()
	if len(params) != 1 || string(params[0][1]) != "{events,missing,users}" {
		t.Errorf("columns query params = %q, want the requested table names", params)
	}
}

func TestGetTablesPicksQueryByTimescaleProbe(t *testing.T) {
	tests := []struct {
		name      string
		timescale string
		want      bool
	}{
		{name: "with timescaledb", timescale: "t", want: true},
		{name: "without timescaledb", timescale: "f", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(tablesResponder(tt.timescale, tableColumns()))

			p, _, _ := newTestPostgres(t, server)

			if _, err := p.GetTables(context.Background(), []domain.TableIdentifier{usersTable()}); err != nil {
				t.Fatalf("GetTables() error = %v", err)
			}

			statement := findStatement(server.recorded(), "tables: columns")
			if got := strings.Contains(statement, "_timescaledb_catalog"); got != tt.want {
				t.Errorf("columns query reads _timescaledb_catalog = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTablesWithoutTablesSkipsTheDatabase(t *testing.T) {
	server := newFakeServer(t)
	p, _, _ := newTestPostgres(t, server)

	got, err := p.GetTables(context.Background(), nil)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("GetTables(nil) = %v, %v, want an empty list", got, err)
	}

	if statements := server.recorded(); len(statements) != 0 {
		t.Errorf("server received %q, want no statements", statements)
	}
}

func TestGetTablesReportsFailures(t *testing.T) {
	tests := []struct {
		name    string
		respond func(string) queryResult
		wantErr string
	}{
		{
			name:    "failed probe",
			respond: failingOn(tablesResponder("f", tableColumns()), "pg_extension", serverError("42501", "permission denied")),
			wantErr: "permission denied",
		},
		{
			name:    "failed columns query",
			respond: failingOn(tablesResponder("f", tableColumns()), "tables: columns", serverError("42501", "permission denied")),
			wantErr: "get table columns",
		},
		{
			name:    "short row",
			respond: tablesResponder("f", queryResult{columns: 2, rows: [][][]byte{{[]byte("public"), []byte("users")}}}),
			wantErr: "expected 10 columns, got 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(tt.respond)

			p, _, _ := newTestPostgres(t, server)

			_, err := p.GetTables(context.Background(), []domain.TableIdentifier{usersTable()})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("GetTables() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetTablesRejectsUnusableConnectionString(t *testing.T) {
	p := New("mysql://u@127.0.0.1/db", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	_, err := p.GetTables(context.Background(), []domain.TableIdentifier{usersTable()})
	if err == nil || !strings.Contains(err.Error(), "unable to connect to database") {
		t.Fatalf("GetTables() error = %v, want it to mention an unusable connection string", err)
	}
}

// tablesResponder answers the timescaledb probe with timescale and the columns
// query with columns.
func tablesResponder(timescale string, columns queryResult) func(string) queryResult {
	return func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "pg_extension"):
			return singleValue(timescale)
		case strings.Contains(sql, "tables: columns"):
			return columns
		default:
			return queryResult{}
		}
	}
}

// tableColumns scripts the result of the columns query.
func tableColumns(rows ...[][]byte) queryResult {
	return queryResult{columns: tablesColumnsWidth, rows: rows}
}

// columnRow is one row of the columns query, its values in query order.
func columnRow(values ...string) [][]byte {
	row := make([][]byte, 0, len(values))
	for _, v := range values {
		row = append(row, []byte(v))
	}

	return row
}

// eventsTable is the public.events table.
func eventsTable() domain.TableIdentifier {
	return domain.TableIdentifier{Schema: "public", Name: "events"}
}

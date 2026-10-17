package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestValidateWhere(t *testing.T) {
	p, _, _ := newTestPostgres(t, newFakeServer(t))

	if err := p.ValidateWhere("id > 1"); err != nil {
		t.Errorf("ValidateWhere(valid) error = %v", err)
	}

	if err := p.ValidateWhere("id > 1; DROP TABLE users"); !errors.Is(err, database.ErrInvalidWhere) {
		t.Errorf("ValidateWhere(invalid) error = %v, want ErrInvalidWhere", err)
	}
}

func TestStartSnapshotExportsLockedSnapshotAndPausesWal(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(nil))
	p, _, _ := newTestPostgres(t, server)

	id, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable(), {Name: "orders"}})
	if err != nil {
		t.Fatalf("StartSnapshot() error = %v", err)
	}
	t.Cleanup(func() { _ = p.FinishSnapshot(context.Background()) })

	if id != testSnapshotID {
		t.Errorf("StartSnapshot() = %q, want %q", id, testSnapshotID)
	}

	statement := findStatement(server.recorded(), "snapshot: export")
	for _, part := range []string{
		"BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;",
		"SET LOCAL lock_timeout = 5000;",
		"SET LOCAL idle_in_transaction_session_timeout = 0;",
		`LOCK TABLE "public"."users", "orders" IN ACCESS SHARE MODE;`,
		"SELECT pg_export_snapshot();",
	} {
		if !strings.Contains(statement, part) {
			t.Errorf("export statement = %q, want it to contain %q", statement, part)
		}
	}

	if !gatePaused(p.gate) {
		t.Error("WAL gate is open, want it paused while the snapshot is active")
	}
}

func TestStartSnapshotReturnsTheActiveSnapshot(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(nil))
	p, _, _ := newTestPostgres(t, server)

	for range 2 {
		id, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable()})
		if err != nil || id != testSnapshotID {
			t.Fatalf("StartSnapshot() = %q, %v, want %q", id, err, testSnapshotID)
		}
	}
	t.Cleanup(func() { _ = p.FinishSnapshot(context.Background()) })

	if exports := countStatements(server.recorded(), "snapshot: export"); exports != 1 {
		t.Fatalf("exports = %d, want 1", exports)
	}
}

func TestStartSnapshotFailureLeavesWalRunning(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(map[string]queryResult{"snapshot: export": serverError("55P03", "lock timeout")}))
	p, _, _ := newTestPostgres(t, server)

	if _, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable()}); !errors.Is(err, database.ErrSnapshot) {
		t.Fatalf("StartSnapshot() error = %v, want ErrSnapshot", err)
	}

	if gatePaused(p.gate) {
		t.Error("WAL gate is paused, want it open after a failed start")
	}
}

func TestStartSnapshotRequiresTables(t *testing.T) {
	p, _, _ := newTestPostgres(t, newFakeServer(t))

	if _, err := p.StartSnapshot(context.Background(), nil); !errors.Is(err, database.ErrSnapshot) {
		t.Fatalf("StartSnapshot() error = %v, want ErrSnapshot", err)
	}
}

func TestStartSnapshotRejectsUnusableConnectionString(t *testing.T) {
	p := New("mysql://u@127.0.0.1/db", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	_, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable()})
	if err == nil || !strings.Contains(err.Error(), "unable to connect to database") {
		t.Fatalf("StartSnapshot() error = %v, want it to mention an unusable connection string", err)
	}

	if gatePaused(p.gate) {
		t.Error("WAL gate is paused, want it open after a failed start")
	}
}

func TestStartSnapshotRejectsEmptySnapshotId(t *testing.T) {
	tests := []struct {
		name   string
		export queryResult
	}{
		{"no rows", queryResult{}},
		{"empty id", singleValue("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(snapshotResponder(map[string]queryResult{"snapshot: export": tt.export}))
			p, _, _ := newTestPostgres(t, server)

			_, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable()})
			if !errors.Is(err, errEmptySnapshotId) || !errors.Is(err, database.ErrSnapshot) {
				t.Fatalf("StartSnapshot() error = %v, want errEmptySnapshotId wrapped in ErrSnapshot", err)
			}

			if gatePaused(p.gate) {
				t.Error("WAL gate is paused, want it open after a failed start")
			}
		})
	}
}

func TestStartSnapshotEndsWithContextWhileWalCallIsInFlight(t *testing.T) {
	server := newFakeServer(t)
	p, _, _ := newTestPostgres(t, server)

	if err := p.gate.enter(context.Background()); err != nil {
		t.Fatalf("enter() error = %v", err)
	}
	t.Cleanup(p.gate.leave)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.StartSnapshot(ctx, []domain.TableIdentifier{usersTable()})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, database.ErrSnapshot) {
		t.Fatalf("StartSnapshot() error = %v, want context.Canceled wrapped in ErrSnapshot", err)
	}

	if gatePaused(p.gate) {
		t.Error("WAL gate is paused, want it open after a failed start")
	}

	if exports := countStatements(server.recorded(), "snapshot: export"); exports != 0 {
		t.Errorf("exports = %d, want none", exports)
	}
}

func TestFinishSnapshotCommitsAndResumesWal(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(nil))
	p, _, _ := newTestPostgres(t, server)

	if _, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable()}); err != nil {
		t.Fatalf("StartSnapshot() error = %v", err)
	}

	for range 2 {
		if err := p.FinishSnapshot(context.Background()); err != nil {
			t.Fatalf("FinishSnapshot() error = %v", err)
		}
	}

	if commits := countStatements(server.recorded(), "COMMIT"); commits != 1 {
		t.Errorf("commits = %d, want 1", commits)
	}

	if gatePaused(p.gate) {
		t.Error("WAL gate is paused, want it open after finish")
	}
}

func TestFinishSnapshotReportsFailedCommit(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(map[string]queryResult{"COMMIT": serverError("25P02", "transaction is aborted")}))
	p, _, _ := newTestPostgres(t, server)

	if _, err := p.StartSnapshot(context.Background(), []domain.TableIdentifier{usersTable()}); err != nil {
		t.Fatalf("StartSnapshot() error = %v", err)
	}

	if err := p.FinishSnapshot(context.Background()); !errors.Is(err, database.ErrSnapshot) {
		t.Fatalf("FinishSnapshot() error = %v, want ErrSnapshot", err)
	}

	if gatePaused(p.gate) {
		t.Error("WAL gate is paused, want it open after a failed commit")
	}

	if err := p.FinishSnapshot(context.Background()); err != nil {
		t.Fatalf("second FinishSnapshot() error = %v, want the snapshot released", err)
	}

	if commits := countStatements(server.recorded(), "COMMIT"); commits != 1 {
		t.Errorf("commits = %d, want 1", commits)
	}
}

func TestReadSnapshotReadsEveryLeaf(t *testing.T) {
	tests := []struct {
		name      string
		timescale bool
		leaves    []snapshotTestLeaf
	}{
		{
			name:   "regular",
			leaves: []snapshotTestLeaf{{name: "users", pages: 3}},
		},
		{
			name:   "partitioned",
			leaves: []snapshotTestLeaf{{name: "users_p1", pages: 1}, {name: "users_p2", pages: 0}},
		},
		{
			name:      "hypertable",
			timescale: true,
			leaves:    []snapshotTestLeaf{{name: "users", pages: 0}, {schema: "_timescaledb_internal", name: "_hyper_1_1_chunk", pages: 7}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(snapshotReadResponder(tt.timescale, tt.leaves))
			p, _, _ := newTestPostgres(t, server)

			var rows []domain.DbRecord
			err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "id > 1", func(record domain.DbRecord) error {
				rows = append(rows, record)
				return nil
			})
			if err != nil {
				t.Fatalf("ReadSnapshot() error = %v", err)
			}

			var want []domain.DbRecord
			for i, leaf := range tt.leaves {
				want = append(want, domain.DbRecord{"id": int32(i + 1), "name": leaf.name})
			}
			if !reflect.DeepEqual(rows, want) {
				t.Fatalf("rows = %#v, want %#v", rows, want)
			}

			statements := server.recorded()

			layout := findStatement(statements, "snapshot: layout")
			if !strings.Contains(layout, "SET TRANSACTION SNAPSHOT '"+testSnapshotID+"';") || !strings.Contains(layout, `'"public"."users"'::regclass`) {
				t.Errorf("layout statement = %q, want it to import the snapshot and name the table", layout)
			}
			if got := strings.Contains(layout, "compressed_chunk_id"); got != tt.timescale {
				t.Errorf("layout skips compressed chunks = %v, want %v", got, tt.timescale)
			}

			for _, leaf := range tt.leaves {
				read := findStatement(statements, "FROM ONLY "+qualifiedTableName(leaf.table()))
				for _, part := range []string{
					"BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;",
					"SET TRANSACTION SNAPSHOT '" + testSnapshotID + "';",
					`COPY (SELECT "id", "name" FROM ONLY`,
					`WHERE (("id" > 1))) TO STDOUT;`,
					"COMMIT;",
				} {
					if !strings.Contains(read, part) {
						t.Errorf("read statement = %q, want it to contain %q", read, part)
					}
				}
			}
		})
	}
}

func TestReadSnapshotSplitsLargeLeavesIntoChunks(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotReadResponder(false, []snapshotTestLeaf{{name: "users", pages: 2*snapshotChunkPages + 1}}))
	p, _, _ := newTestPostgres(t, server)

	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error { return nil })
	if err != nil {
		t.Fatalf("ReadSnapshot() error = %v", err)
	}

	var reads []string
	for _, statement := range server.recorded() {
		if strings.Contains(statement, "snapshot: read") {
			reads = append(reads, statement)
		}
	}

	wants := []string{
		`WHERE ctid < '(1024,0)'::tid AND (true)`,
		`WHERE ctid >= '(1024,0)'::tid AND ctid < '(2048,0)'::tid AND (true)`,
		`WHERE ctid >= '(2048,0)'::tid AND (true)`,
	}
	if len(reads) != len(wants) {
		t.Fatalf("reads = %d, want %d", len(reads), len(wants))
	}
	for i, want := range wants {
		if !strings.Contains(reads[i], want) {
			t.Errorf("read %d = %q, want it to contain %q", i, reads[i], want)
		}
	}
}

func TestReadSnapshotObservesProcessedRowsAsInserts(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotReadResponder(false, []snapshotTestLeaf{{name: "users_p1", pages: 1}, {name: "users_p2", pages: 1}}))
	p, _, _ := newTestPostgres(t, server)

	calls := 0
	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error {
		calls++
		if calls == 2 {
			return errors.New("stop")
		}
		return nil
	})
	if err == nil {
		t.Fatal("ReadSnapshot() error = nil, want the row error")
	}

	if got := observedRows(t, p, domain.OperationTypeInsert, usersTable()); got != 1 {
		t.Fatalf("observed inserts = %d, want only the processed row", got)
	}
}

func TestReadSnapshotStopsOnRowError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotReadResponder(false, []snapshotTestLeaf{{name: "users_p1", pages: 1}, {name: "users_p2", pages: 1}}))
	p, _, _ := newTestPostgres(t, server)

	stop := errors.New("stop")
	calls := 0
	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error {
		calls++
		return stop
	})

	if !errors.Is(err, stop) {
		t.Fatalf("ReadSnapshot() error = %v, want the row error", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want the read to stop at the first row", calls)
	}
}

func TestReadSnapshotRejectsInvalidWhereWithoutQuerying(t *testing.T) {
	server := newFakeServer(t)
	p, _, _ := newTestPostgres(t, server)

	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "1; DROP TABLE users", func(domain.DbRecord) error { return nil })
	if !errors.Is(err, database.ErrInvalidWhere) {
		t.Fatalf("ReadSnapshot() error = %v, want ErrInvalidWhere", err)
	}

	if statements := server.recorded(); len(statements) != 0 {
		t.Fatalf("statements = %q, want none", statements)
	}
}

func TestReadSnapshotFailsWhenSnapshotIsGone(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(map[string]queryResult{
		"pg_extension":     singleValue("f"),
		"snapshot: layout": serverError("22023", "invalid snapshot identifier"),
	}))
	p, _, _ := newTestPostgres(t, server)

	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error { return nil })
	if !errors.Is(err, database.ErrSnapshot) {
		t.Fatalf("ReadSnapshot() error = %v, want ErrSnapshot", err)
	}
}

func TestReadSnapshotRejectsUnusableConnectionString(t *testing.T) {
	p := New("mysql://u@127.0.0.1/db", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "unable to connect to database") {
		t.Fatalf("ReadSnapshot() error = %v, want it to mention an unusable connection string", err)
	}
}

func TestReadSnapshotFailsWhenExtensionProbeFails(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(snapshotResponder(map[string]queryResult{"pg_extension": serverError("42501", "permission denied")}))
	p, _, _ := newTestPostgres(t, server)

	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("ReadSnapshot() error = %v, want the probe failure", err)
	}

	if layouts := countStatements(server.recorded(), "snapshot: layout"); layouts != 0 {
		t.Errorf("layout queries = %d, want none", layouts)
	}
}

func TestReadSnapshotRejectsUnusableLayout(t *testing.T) {
	tests := []struct {
		name    string
		layout  queryResult
		wantErr string
	}{
		{"missing results", queryResult{}, "expected 5 results, got 1"},
		{"no columns", scriptedSnapshotLayout([][][]byte{{[]byte("public"), []byte("users"), []byte("1")}}, nil), "has no columns to read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(snapshotResponder(map[string]queryResult{
				"pg_extension":     singleValue("f"),
				"snapshot: layout": tt.layout,
			}))
			p, _, _ := newTestPostgres(t, server)

			err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error { return nil })
			if !errors.Is(err, database.ErrSnapshot) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ReadSnapshot() error = %v, want ErrSnapshot mentioning %q", err, tt.wantErr)
			}

			if reads := countStatements(server.recorded(), "snapshot: read"); reads != 0 {
				t.Errorf("reads = %d, want none", reads)
			}
		})
	}
}

func TestReadSnapshotReportsUnterminatedRow(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(failingOn(
		snapshotReadResponder(false, []snapshotTestLeaf{{name: "users", pages: 1}}),
		"snapshot: read",
		queryResult{copyOut: [][]byte{[]byte("1\tusers")}},
	))
	p, _, _ := newTestPostgres(t, server)

	calls := 0
	err := p.ReadSnapshot(context.Background(), testSnapshotID, usersTable(), "", func(domain.DbRecord) error {
		calls++
		return nil
	})

	if !errors.Is(err, errCopyRow) {
		t.Fatalf("ReadSnapshot() error = %v, want errCopyRow", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want the unterminated row held back", calls)
	}
}

func TestParseSnapshotLayoutRejectsMalformedRows(t *testing.T) {
	leaf := [][]byte{[]byte("public"), []byte("users"), []byte("1")}
	column := [][]byte{[]byte("id"), []byte("23")}

	tests := []struct {
		name    string
		leaves  [][][]byte
		columns [][][]byte
		wantErr string
	}{
		{"short leaf row", [][][]byte{leaf[:2]}, [][][]byte{column}, "expected 3 leaf columns, got 2"},
		{"unparseable pages", [][][]byte{{leaf[0], leaf[1], []byte("many")}}, [][][]byte{column}, "parse pages"},
		{"short column row", [][][]byte{leaf}, [][][]byte{column[:1]}, "expected 2 column fields, got 1"},
		{"unparseable column type", [][][]byte{leaf}, [][][]byte{{column[0], []byte("int4")}}, "parse column type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&snapshot{}).parseSnapshotLayout(&pgconn.Result{Rows: tt.leaves}, &pgconn.Result{Rows: tt.columns})
			if !errors.Is(err, database.ErrSnapshot) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseSnapshotLayout() error = %v, want ErrSnapshot mentioning %q", err, tt.wantErr)
			}
		})
	}
}

func TestSnapshotRanges(t *testing.T) {
	tests := []struct {
		pages uint64
		want  []string
	}{
		{0, []string{""}},
		{snapshotChunkPages, []string{""}},
		{snapshotChunkPages + 1, []string{
			"ctid < '(1024,0)'::tid AND ",
			"ctid >= '(1024,0)'::tid AND ",
		}},
	}

	for _, tt := range tests {
		if got := (&snapshot{}).snapshotRanges(tt.pages); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("snapshotRanges(%d) = %q, want %q", tt.pages, got, tt.want)
		}
	}
}

// testSnapshotID is the id the fake server exports.
const testSnapshotID = "00000003-0000001B-1"

// snapshotTestLeaf is one leaf relation the fake layout query reports.
type snapshotTestLeaf struct {
	schema string
	name   string
	pages  uint64
}

// table returns the leaf's identifier, in public unless a schema is set.
func (l snapshotTestLeaf) table() domain.TableIdentifier {
	schema := l.schema
	if schema == "" {
		schema = "public"
	}

	return domain.TableIdentifier{Schema: domain.TableSchema(schema), Name: domain.TableName(l.name)}
}

// snapshotResponder answers the snapshot export with testSnapshotID and any
// statement containing a key of overrides with its value.
func snapshotResponder(overrides map[string]queryResult) func(string) queryResult {
	return func(sql string) queryResult {
		for marker, result := range overrides {
			if strings.Contains(sql, marker) {
				return result
			}
		}

		if strings.Contains(sql, "snapshot: export") {
			return singleValue(testSnapshotID)
		}

		return queryResult{}
	}
}

// snapshotReadResponder answers the timescaledb probe with timescale, the
// layout query with leaves and an int4 id and text name column, and the read
// of the i-th leaf with one row: id i+1, name the leaf's name.
func snapshotReadResponder(timescale bool, leaves []snapshotTestLeaf) func(string) queryResult {
	flag := "f"
	if timescale {
		flag = "t"
	}

	leafRows := make([][][]byte, len(leaves))
	for i, leaf := range leaves {
		table := leaf.table()
		leafRows[i] = [][]byte{[]byte(table.Schema), []byte(table.Name), fmt.Append(nil, leaf.pages)}
	}

	return func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "pg_extension"):
			return singleValue(flag)
		case strings.Contains(sql, "snapshot: layout"):
			return queryResult{statements: []queryResult{
				{},
				{},
				{columns: 3, rows: leafRows},
				{columns: 2, rows: [][][]byte{{[]byte("id"), []byte("23")}, {[]byte("name"), []byte("25")}}},
				{},
			}}
		case strings.Contains(sql, "snapshot: read"):
			for i, leaf := range leaves {
				if strings.Contains(sql, "FROM ONLY "+qualifiedTableName(leaf.table())+" ") {
					return queryResult{copyOut: [][]byte{fmt.Appendf(nil, "%d\t%s\n", i+1, leaf.name)}}
				}
			}
			return queryResult{err: &pgproto3.ErrorResponse{Severity: "ERROR", Code: "42P01", Message: "unknown leaf"}}
		default:
			return queryResult{}
		}
	}
}

// scriptedSnapshotLayout scripts the five results of the layout query around
// the given leaf and column rows.
func scriptedSnapshotLayout(leaves, columns [][][]byte) queryResult {
	return queryResult{statements: []queryResult{
		{},
		{},
		{columns: 3, rows: leaves},
		{columns: 2, rows: columns},
		{},
	}}
}

// observedRows returns how many rows of operation on table p's processed-rows
// histogram has observed.
func observedRows(t *testing.T, p *Postgres, operation domain.OperationType, table domain.TableIdentifier) uint64 {
	t.Helper()

	var metric dto.Metric
	observer := p.metrics.rowDurations.WithLabelValues(string(operation), string(table.Schema), string(table.Name))
	if err := observer.(prometheus.Metric).Write(&metric); err != nil {
		t.Fatalf("write histogram: %v", err)
	}

	return metric.GetHistogram().GetSampleCount()
}

// countStatements returns how many statements contain marker.
func countStatements(statements []string, marker string) int {
	n := 0
	for _, statement := range statements {
		if strings.Contains(statement, marker) {
			n++
		}
	}

	return n
}

// gatePaused reports whether gate is paused.
func gatePaused(gate *walGate) bool {
	gate.mu.Lock()
	defer gate.mu.Unlock()

	return gate.paused
}

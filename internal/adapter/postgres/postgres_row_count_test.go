package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestGetRowCountStreamsEstimateThenExactCount(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(rowCountResponder(singleValue("t"), planEstimate(42), singleValue("40")))

	p, _, _ := newTestPostgres(t, server)

	counts, err := p.GetRowCount(context.Background(), usersTable(), "id > 1")
	if err != nil {
		t.Fatalf("GetRowCount() error = %v", err)
	}

	got := drainRowCounts(t, counts)
	want := []database.RowCount{{Num: 42, Rough: true}, {Num: 40}}
	if !equalRowCounts(got, want) {
		t.Fatalf("GetRowCount() streamed %+v, want %+v", got, want)
	}

	if params := server.boundParams(); len(params) != 1 || string(params[0][0]) != `"public"."users"` {
		t.Errorf("table exists params = %q, want the quoted qualified name", params)
	}

	statements := server.recorded()
	for _, marker := range []string{"row count: planner estimate", "row count: exact"} {
		statement := findStatement(statements, marker)

		for _, part := range []string{
			"BEGIN READ ONLY;",
			"SET LOCAL statement_timeout = 2000;",
			`FROM "public"."users" WHERE ("id" > 1);`,
			"ROLLBACK;",
		} {
			if !strings.Contains(statement, part) {
				t.Errorf("%s statement = %q, want it to contain %q", marker, statement, part)
			}
		}
	}

	if statement := findStatement(statements, "row count: planner estimate"); !strings.Contains(statement, "EXPLAIN (FORMAT JSON) SELECT 1 FROM") {
		t.Errorf("estimate statement = %q, want an EXPLAIN of the filtered select", statement)
	}

	if statement := findStatement(statements, "row count: exact"); !strings.Contains(statement, "SELECT count(*) FROM") {
		t.Errorf("exact statement = %q, want a count of the filtered table", statement)
	}
}

func TestGetRowCountWithoutFilterOrSchema(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(rowCountResponder(singleValue("t"), planEstimate(7), singleValue("7")))

	p, _, _ := newTestPostgres(t, server)

	counts, err := p.GetRowCount(context.Background(), domain.TableIdentifier{Name: "Users"}, "  ")
	if err != nil {
		t.Fatalf("GetRowCount() error = %v", err)
	}

	drainRowCounts(t, counts)

	if params := server.boundParams(); len(params) != 1 || string(params[0][0]) != `"Users"` {
		t.Errorf("table exists params = %q, want the quoted bare name", params)
	}

	if statement := findStatement(server.recorded(), "row count: exact"); !strings.Contains(statement, `SELECT count(*) FROM "Users";`) {
		t.Errorf("exact statement = %q, want an unfiltered count", statement)
	}
}

func TestGetRowCountRejectsInvalidWhereWithoutQuerying(t *testing.T) {
	server := newFakeServer(t)
	p, _, _ := newTestPostgres(t, server)

	counts, err := p.GetRowCount(context.Background(), usersTable(), "1 = 1; DROP TABLE users")
	if !errors.Is(err, database.ErrInvalidWhere) {
		t.Fatalf("GetRowCount() error = %v, want ErrInvalidWhere", err)
	}

	if counts != nil {
		t.Error("GetRowCount() returned a channel alongside the error")
	}

	if statements := server.recorded(); len(statements) != 0 {
		t.Errorf("server received %q, want no statements", statements)
	}
}

func TestGetRowCountRejectsUnusableConnectionString(t *testing.T) {
	p := New("mysql://u@127.0.0.1/db", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	_, err := p.GetRowCount(context.Background(), usersTable(), "")
	if err == nil || !strings.Contains(err.Error(), "unable to connect to database") {
		t.Fatalf("GetRowCount() error = %v, want it to mention an unusable connection string", err)
	}
}

func TestGetRowCountFailsBeforeStreaming(t *testing.T) {
	tests := []struct {
		name    string
		exists  queryResult
		wantErr error
		wantMsg string
	}{
		{name: "missing table", exists: singleValue("f"), wantErr: database.ErrTableNotFound, wantMsg: `table "public"."users"`},
		{name: "lookup error", exists: serverError("42501", "permission denied"), wantMsg: "permission denied"},
		{name: "empty lookup", exists: queryResult{columns: 1}, wantErr: errEmptyRowCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(rowCountResponder(tt.exists, planEstimate(1), singleValue("1")))

			p, _, _ := newTestPostgres(t, server)

			counts, err := p.GetRowCount(context.Background(), usersTable(), "")
			if err == nil || counts != nil {
				t.Fatalf("GetRowCount() = (%v, %v), want only an error", counts, err)
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("GetRowCount() error = %v, want %v", err, tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("GetRowCount() error = %v, want it to mention %q", err, tt.wantMsg)
			}

			if statement := findStatement(server.recorded(), "row count: planner estimate"); statement != "" {
				t.Errorf("server received %q after a failed lookup", statement)
			}
		})
	}
}

func TestGetRowCountSendsEstimateErrorAndSkipsExactCount(t *testing.T) {
	tests := []struct {
		name     string
		estimate queryResult
		wantMsg  string
	}{
		{name: "server error", estimate: serverError("57014", "canceling statement due to statement timeout"), wantMsg: "statement timeout"},
		{name: "malformed plan", estimate: singleValue("not json"), wantMsg: "decode plan"},
		{name: "empty plan", estimate: singleValue("[]"), wantMsg: errEmptyRowCount.Error()},
		{name: "no rows", estimate: queryResult{columns: 1}, wantMsg: errEmptyRowCount.Error()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(rowCountResponder(singleValue("t"), tt.estimate, singleValue("1")))

			p, _, _ := newTestPostgres(t, server)

			counts, err := p.GetRowCount(context.Background(), usersTable(), "")
			if err != nil {
				t.Fatalf("GetRowCount() error = %v", err)
			}

			got := drainRowCounts(t, counts)
			if len(got) != 1 || got[0].Error == nil || !strings.Contains(got[0].Error.Error(), tt.wantMsg) {
				t.Fatalf("GetRowCount() streamed %+v, want one error mentioning %q", got, tt.wantMsg)
			}

			if statement := findStatement(server.recorded(), "row count: exact"); statement != "" {
				t.Errorf("server received %q after a failed estimate", statement)
			}
		})
	}
}

func TestGetRowCountDropsFailedExactCount(t *testing.T) {
	tests := []struct {
		name  string
		exact queryResult
	}{
		{name: "statement timeout", exact: serverError("57014", "canceling statement due to statement timeout")},
		{name: "other server error", exact: serverError("42703", "column does not exist")},
		{name: "malformed count", exact: singleValue("many")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(rowCountResponder(singleValue("t"), planEstimate(1000), tt.exact))

			p, _, _ := newTestPostgres(t, server)

			counts, err := p.GetRowCount(context.Background(), usersTable(), "")
			if err != nil {
				t.Fatalf("GetRowCount() error = %v", err)
			}

			got := drainRowCounts(t, counts)
			want := []database.RowCount{{Num: 1000, Rough: true}}
			if !equalRowCounts(got, want) {
				t.Fatalf("GetRowCount() streamed %+v, want %+v", got, want)
			}
		})
	}
}

func TestGetRowCountRunsConcurrentCallsIndependently(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "row count: table exists"):
			return singleValue("t")
		case strings.Contains(sql, "row count: planner estimate"):
			return planEstimate(tableNumber(sql) * 10)
		case strings.Contains(sql, "row count: exact"):
			return singleValue(fmt.Sprint(tableNumber(sql)))
		default:
			return queryResult{}
		}
	})

	p, _, _ := newTestPostgres(t, server)

	const calls = 8

	var wg sync.WaitGroup
	results := make([][]database.RowCount, calls)
	errs := make([]error, calls)

	for i := range calls {
		wg.Go(func() {
			table := domain.TableIdentifier{Schema: "public", Name: domain.TableName(fmt.Sprintf("t%d", i))}

			counts, err := p.GetRowCount(context.Background(), table, "")
			if err != nil {
				errs[i] = err
				return
			}

			results[i] = drainRowCounts(t, counts)
		})
	}
	wg.Wait()

	for i := range calls {
		if errs[i] != nil {
			t.Fatalf("GetRowCount(t%d) error = %v", i, errs[i])
		}

		want := []database.RowCount{{Num: uint64(i * 10), Rough: true}, {Num: uint64(i)}}
		if !equalRowCounts(results[i], want) {
			t.Errorf("GetRowCount(t%d) streamed %+v, want %+v", i, results[i], want)
		}
	}
}

// rowCountResponder answers the three row count statements with the given
// results, leaving every other statement a bare success.
func rowCountResponder(exists, estimate, exact queryResult) func(string) queryResult {
	return func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "row count: table exists"):
			return exists
		case strings.Contains(sql, "row count: planner estimate"):
			return estimate
		case strings.Contains(sql, "row count: exact"):
			return exact
		default:
			return queryResult{}
		}
	}
}

// planEstimate scripts an EXPLAIN (FORMAT JSON) result whose top node expects rows.
func planEstimate(rows int) queryResult {
	return singleValue(fmt.Sprintf(`[{"Plan": {"Node Type": "Seq Scan", "Plan Rows": %d}}]`, rows))
}

// usersTable is the table most row count tests ask about.
func usersTable() domain.TableIdentifier {
	return domain.TableIdentifier{Schema: "public", Name: "users"}
}

// tableNumber extracts n from the "public"."tn" table a row count statement names.
func tableNumber(sql string) int {
	var n int
	_, _ = fmt.Sscanf(sql[strings.Index(sql, `"public"."t`)+len(`"public"."t`):], "%d", &n)

	return n
}

// drainRowCounts reads counts until it is closed, failing the test if that
// takes longer than five seconds.
func drainRowCounts(t testing.TB, counts <-chan database.RowCount) []database.RowCount {
	t.Helper()

	var got []database.RowCount
	timeout := time.After(5 * time.Second)

	for {
		select {
		case count, ok := <-counts:
			if !ok {
				return got
			}
			got = append(got, count)
		case <-timeout:
			t.Errorf("timed out waiting for the row count stream to close")
			return got
		}
	}
}

// equalRowCounts reports whether got and want hold the same error-free counts.
func equalRowCounts(got, want []database.RowCount) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i].Error != nil || got[i].Num != want[i].Num || got[i].Rough != want[i].Rough {
			return false
		}
	}

	return true
}

// findStatement returns the first statement containing marker, or "".
func findStatement(statements []string, marker string) string {
	for _, statement := range statements {
		if strings.Contains(statement, marker) {
			return statement
		}
	}

	return ""
}

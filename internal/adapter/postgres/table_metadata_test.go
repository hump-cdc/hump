package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestGetTablesMetadataRejectsEmptyTableList(t *testing.T) {
	server := newFakeServer(t)
	loader := newTestTableMetadataLoader(t, server)

	_, err := loader.getTablesMetadata(context.Background(), nil)
	if !errors.Is(err, database.ErrNoTablesConfigured) {
		t.Fatalf("getTablesMetadata() error = %v, want database.ErrNoTablesConfigured", err)
	}

	if statements := server.recorded(); len(statements) != 0 {
		t.Errorf("statements = %q, want none: an empty list must not reach the server", statements)
	}
}

func TestGetTablesMetadataBindsTablesAsArrays(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
		layoutRow("other", "events", tableTypeRegular, nil, nil),
	))
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "other", Name: "events"},
	}

	if _, err := loader.getTablesMetadata(context.Background(), tables); err != nil {
		t.Fatalf("getTablesMetadata() error = %v", err)
	}

	params := server.boundParams()
	if len(params) != 1 {
		t.Fatalf("binds = %d, want 1", len(params))
	}
	if len(params[0]) != 2 {
		t.Fatalf("bound parameters = %d, want 2", len(params[0]))
	}
	if got := string(params[0][0]); got != "{public,other}" {
		t.Errorf("schemas parameter = %q, want %q", got, "{public,other}")
	}
	if got := string(params[0][1]); got != "{users,events}" {
		t.Errorf("names parameter = %q, want %q", got, "{users,events}")
	}
}

func TestGetTablesMetadataSelectsQueryByTimescaleDB(t *testing.T) {
	cases := []struct {
		name      string
		timescale bool
		want      string
	}{
		{name: "without timescaledb", timescale: false, want: tableMetadataTablesNoTimescaleSql},
		{name: "with timescaledb", timescale: true, want: tableMetadataTablesTimescaleSql},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(tableLayoutResponder(tt.timescale,
				layoutRow("public", "users", tableTypeRegular, nil, nil),
			))
			loader := newTestTableMetadataLoader(t, server)

			tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}
			if _, err := loader.getTablesMetadata(context.Background(), tables); err != nil {
				t.Fatalf("getTablesMetadata() error = %v", err)
			}

			statements := server.recorded()
			if len(statements) != 2 {
				t.Fatalf("statements = %d, want 2: %q", len(statements), statements)
			}
			if statements[1] != tt.want {
				t.Errorf("layout query = %q, want the other variant", statements[1])
			}
		})
	}
}

func TestGetTablesMetadataGroupsPartitionsByTable(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(true,
		layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
		layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2025")),
		layoutRow("public", "metrics", tableTypeHypertable, []byte("_timescaledb_internal"), nil),
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "events"},
		{Schema: "public", Name: "metrics"},
		{Schema: "public", Name: "users"},
	}

	info, err := loader.getTablesMetadata(context.Background(), tables)
	if err != nil {
		t.Fatalf("getTablesMetadata() error = %v", err)
	}

	events := info[domain.TableIdentifier{Schema: "public", Name: "events"}]
	if events.Type != tableTypePartitioned {
		t.Errorf("events type = %v, want %v", events.Type, tableTypePartitioned)
	}
	wantPartitions := []domain.TableIdentifier{
		{Schema: "public", Name: "events_2024"},
		{Schema: "public", Name: "events_2025"},
	}
	if len(events.Partitions) != len(wantPartitions) {
		t.Fatalf("events partitions = %+v, want %+v", events.Partitions, wantPartitions)
	}
	for i, want := range wantPartitions {
		if events.Partitions[i] != want {
			t.Errorf("events partition %d = %+v, want %+v", i, events.Partitions[i], want)
		}
	}

	metrics := info[domain.TableIdentifier{Schema: "public", Name: "metrics"}]
	if len(metrics.Partitions) != 1 || metrics.Partitions[0].Schema != "_timescaledb_internal" {
		t.Fatalf("metrics partitions = %+v, want one in _timescaledb_internal", metrics.Partitions)
	}
	if metrics.Partitions[0].Name != "" {
		t.Errorf("metrics partition name = %q, want empty: the column was NULL", metrics.Partitions[0].Name)
	}

	if partitions := info[domain.TableIdentifier{Schema: "public", Name: "users"}].Partitions; partitions != nil {
		t.Errorf("users partitions = %+v, want none", partitions)
	}
}

func TestGetTablesMetadataKeepsEmptyPartitionSchema(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "events", tableTypePartitioned, []byte(""), []byte("events_2024")),
	))
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "events"}}

	info, err := loader.getTablesMetadata(context.Background(), tables)
	if err != nil {
		t.Fatalf("getTablesMetadata() error = %v", err)
	}

	partitions := info[domain.TableIdentifier{Schema: "public", Name: "events"}].Partitions
	want := domain.TableIdentifier{Schema: "", Name: "events_2024"}
	if len(partitions) != 1 || partitions[0] != want {
		t.Fatalf("partitions = %+v, want %+v: an empty schema is not a NULL one", partitions, want)
	}
}

func TestGetTablesMetadataReportsMissingTables(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "gone"},
		{Schema: "audit", Name: "trail"},
	}

	_, err := loader.getTablesMetadata(context.Background(), tables)
	if !errors.Is(err, database.ErrTablesNotFound) {
		t.Fatalf("getTablesMetadata() error = %v, want database.ErrTablesNotFound", err)
	}

	if !strings.Contains(err.Error(), "public.gone, audit.trail") {
		t.Errorf("error = %v, want it to list every missing table", err)
	}
}

func TestGetTablesMetadataRejectsShortRows(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(sql string) queryResult {
		if strings.Contains(sql, "pg_extension") {
			return singleValue("f")
		}

		return tableLayout([][]byte{[]byte("public"), []byte("users"), []byte("regular")})
	})
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := loader.getTablesMetadata(context.Background(), tables)
	if err == nil {
		t.Fatal("getTablesMetadata() error = nil, want the column count error")
	}
	if !strings.Contains(err.Error(), "expected 5 columns, got 3") {
		t.Errorf("error = %v, want it to report the column count", err)
	}
}

func TestGetTablesMetadataWrapsQueryError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(sql string) queryResult {
		if strings.Contains(sql, "pg_extension") {
			return singleValue("f")
		}

		return serverError("42P01", "relation does not exist")
	})
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := loader.getTablesMetadata(context.Background(), tables)
	if err == nil {
		t.Fatal("getTablesMetadata() error = nil, want the server error")
	}
	if !strings.Contains(err.Error(), "get tables with partitions") {
		t.Errorf("error = %v, want it wrapped by the lookup", err)
	}
}

func TestGetTablesMetadataPropagatesTimescaleProbeError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	loader := newTestTableMetadataLoader(t, server)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := loader.getTablesMetadata(context.Background(), tables)
	if err == nil {
		t.Fatal("getTablesMetadata() error = nil, want the probe error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %v, want the probe error", err)
	}

	if statements := server.recorded(); len(statements) != 1 {
		t.Errorf("statements = %d, want 1: the layout query must not run", len(statements))
	}
}

func TestGetTablesMetadataWrapsEncodeError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	loader := newTestTableMetadataLoader(t, server)
	loader.newTypeMap = newUnencodableTypeMap

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := loader.getTablesMetadata(context.Background(), tables)
	if err == nil {
		t.Fatal("getTablesMetadata() error = nil, want the encode error")
	}
	if !strings.Contains(err.Error(), "encode schemas array") {
		t.Errorf("error = %v, want it to name the array that failed", err)
	}

	if statements := server.recorded(); len(statements) != 1 {
		t.Errorf("statements = %d, want 1: the layout query must not run", len(statements))
	}
}

func TestHasTimescaleDB(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "installed", value: "t", want: true},
		{name: "absent", value: "f", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(func(string) queryResult { return singleValue(tt.value) })
			loader := newTestTableMetadataLoader(t, server)

			got, err := loader.hasTimescaleDB(context.Background())
			if err != nil {
				t.Fatalf("hasTimescaleDB() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("hasTimescaleDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasTimescaleDBProbesOnce(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return singleValue("t") })
	loader := newTestTableMetadataLoader(t, server)

	for range 3 {
		got, err := loader.hasTimescaleDB(context.Background())
		if err != nil {
			t.Fatalf("hasTimescaleDB() error = %v", err)
		}
		if !got {
			t.Error("hasTimescaleDB() = false, want the cached true")
		}
	}

	if statements := server.recorded(); len(statements) != 1 {
		t.Errorf("statements = %q, want one probe", statements)
	}
}

func TestHasTimescaleDBRejectsEmptyResult(t *testing.T) {
	cases := []struct {
		name   string
		result queryResult
	}{
		{name: "no results", result: queryResult{empty: true}},
		{name: "no rows", result: queryResult{columns: 1}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(func(string) queryResult { return tt.result })
			loader := newTestTableMetadataLoader(t, server)

			_, err := loader.hasTimescaleDB(context.Background())
			if !errors.Is(err, database.ErrDbSetup) {
				t.Fatalf("hasTimescaleDB() error = %v, want ErrDbSetup", err)
			}
		})
	}
}

func TestHasTimescaleDBPropagatesServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	loader := newTestTableMetadataLoader(t, server)

	if _, err := loader.hasTimescaleDB(context.Background()); err == nil {
		t.Fatal("hasTimescaleDB() error = nil, want the server error")
	}
}

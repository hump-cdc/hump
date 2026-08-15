package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestNewIntrospectorBorrowsConnection(t *testing.T) {
	server := newFakeServer(t)
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	if introspector.connection == nil {
		t.Fatal("newIntrospector() left the connection nil")
	}
	if introspector.registry == nil {
		t.Fatal("newIntrospector() left the registry nil")
	}
	if introspector.initialized {
		t.Error("newIntrospector() marked the introspector initialized before the first introspection")
	}
}

func TestInvalidateRefreshesPartitions(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
	))
	ensure, calls := replicaIdentityRecorder()
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, ensure)

	if err := introspector.Invalidate(context.Background(), nil); err != nil {
		t.Fatalf("Invalidate() error = %v", err)
	}

	partitions := introspector.registry.Partitions()
	partition := domain.TableIdentifier{Schema: "public", Name: "events_2024"}
	if parent, ok := partitions[partition]; !ok || parent.Name != "events" {
		t.Fatalf("Partitions() = %+v, want %+v mapped to public.events", partitions, partition)
	}

	if len(*calls) != 1 {
		t.Fatalf("ensureReplicaIdentity calls = %d, want 1", len(*calls))
	}
	got := (*calls)[0]
	want := []domain.TableIdentifier{
		{Schema: "public", Name: "events"},
		{Schema: "public", Name: "events_2024"},
	}
	if len(got) != len(want) {
		t.Fatalf("ensureReplicaIdentity tables = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ensureReplicaIdentity table %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestInvalidateMarksInitializedOnlyForFullRefresh(t *testing.T) {
	cases := []struct {
		name  string
		table *domain.TableIdentifier
		want  bool
	}{
		{name: "full refresh", table: nil, want: true},
		{name: "narrowed to a partition", table: &domain.TableIdentifier{Schema: "public", Name: "events_2024"}, want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(tableLayoutResponder(false,
				layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
			))
			introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

			if err := introspector.Invalidate(context.Background(), tt.table); err != nil {
				t.Fatalf("Invalidate() error = %v", err)
			}

			if introspector.initialized != tt.want {
				t.Errorf("initialized = %v, want %v", introspector.initialized, tt.want)
			}
		})
	}
}

func TestInvalidateSkipsRefreshForUnknownTable(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
	))
	ensure, calls := replicaIdentityRecorder()
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, ensure)

	table := domain.TableIdentifier{Schema: "other", Name: "logs"}
	if err := introspector.Invalidate(context.Background(), &table); err != nil {
		t.Fatalf("Invalidate() error = %v", err)
	}

	if statements := server.recorded(); len(statements) != 2 {
		t.Errorf("statements = %d, want 2: only the metadata lookup must run", len(statements))
	}
	if len(*calls) != 0 {
		t.Errorf("ensureReplicaIdentity calls = %d, want 0", len(*calls))
	}
	if partitions := introspector.registry.Partitions(); len(partitions) != 0 {
		t.Errorf("Partitions() = %+v, want none: the refresh must not have run", partitions)
	}
}

func TestInvalidatePropagatesPartitionProbeError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	table := domain.TableIdentifier{Schema: "public", Name: "events_2024"}
	err := introspector.Invalidate(context.Background(), &table)
	if err == nil {
		t.Fatal("Invalidate() error = nil, want the probe error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %v, want the probe error", err)
	}
}

func TestInvalidatePropagatesIntrospectionError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	err := introspector.Invalidate(context.Background(), nil)
	if err == nil {
		t.Fatal("Invalidate() error = nil, want the introspection error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %v, want the introspection error", err)
	}
}

func TestInvalidatePropagatesReplicaIdentityError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))

	wantErr := errors.New("cannot alter replica identity")
	var got []domain.TableIdentifier
	ensure := func(_ context.Context, _ *connection, tables []domain.TableIdentifier) error {
		got = tables

		return wantErr
	}
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, ensure)

	if err := introspector.Invalidate(context.Background(), nil); !errors.Is(err, wantErr) {
		t.Fatalf("Invalidate() error = %v, want %v", err, wantErr)
	}

	want := introspector.registry.ReplicaIdentityFullTables()
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("ensureReplicaIdentity tables = %+v, want %+v", got, want)
	}
	if introspector.initialized {
		t.Error("initialized = true, want false: a failed refresh must not count as initialized")
	}
}

func TestResolveTableConfigurationIntrospectsOnFirstCall(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	table := domain.TableIdentifier{Schema: "public", Name: "users"}
	ref, err := introspector.ResolveTableConfiguration(context.Background(), table)
	if err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}
	if ref != table {
		t.Errorf("table = %+v, want %+v", ref, table)
	}

	if statements := server.recorded(); len(statements) != 2 {
		t.Errorf("statements = %d, want 2: the probe and the layout query", len(statements))
	}
}

func TestResolveTableConfigurationIntrospectsOnce(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	table := domain.TableIdentifier{Schema: "public", Name: "users"}
	for range 3 {
		if _, err := introspector.ResolveTableConfiguration(context.Background(), table); err != nil {
			t.Fatalf("ResolveTableConfiguration() error = %v", err)
		}
	}

	if statements := server.recorded(); len(statements) != 2 {
		t.Errorf("statements = %d, want 2: a hit must not re-introspect", len(statements))
	}
}

func TestResolveTableConfigurationPropagatesInitialIntrospectionError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	table := domain.TableIdentifier{Schema: "public", Name: "users"}
	ref, err := introspector.ResolveTableConfiguration(context.Background(), table)
	if err == nil {
		t.Fatal("ResolveTableConfiguration() error = nil, want the introspection error")
	}
	if ref != (domain.TableIdentifier{}) {
		t.Errorf("ResolveTableConfiguration() = %+v, want the zero table", ref)
	}
}

func TestResolveTableConfigurationResolvesKnownPartition(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	partition := domain.TableIdentifier{Schema: "public", Name: "events_2024"}
	ref, err := introspector.ResolveTableConfiguration(context.Background(), partition)
	if err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}

	want := domain.TableIdentifier{Schema: "public", Name: "events"}
	if ref != want {
		t.Errorf("table = %+v, want the parent %+v", ref, want)
	}

	if statements := server.recorded(); len(statements) != 2 {
		t.Errorf("statements = %d, want 2: the first introspection already knew the partition", len(statements))
	}
}

func TestResolveTableConfigurationReIntrospectsNewPartition(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(stagedLayoutResponder(
		[][][]byte{layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024"))},
		[][][]byte{
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2025")),
		},
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	partition := domain.TableIdentifier{Schema: "public", Name: "events_2025"}
	ref, err := introspector.ResolveTableConfiguration(context.Background(), partition)
	if err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}

	want := domain.TableIdentifier{Schema: "public", Name: "events"}
	if ref != want {
		t.Errorf("table = %+v, want the parent %+v", ref, want)
	}

	if _, ignored := introspector.ignoredTables[partition]; ignored {
		t.Error("a partition resolved by the retry must not be ignored")
	}
}

func TestResolveTableConfigurationCostsOneQueryPerMiss(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(stagedLayoutResponder(
		[][][]byte{layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024"))},
		[][][]byte{
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2025")),
		},
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	known := domain.TableIdentifier{Schema: "public", Name: "events_2024"}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), known); err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}

	before := len(server.recorded())
	partition := domain.TableIdentifier{Schema: "public", Name: "events_2025"}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), partition); err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}

	statements := server.recorded()[before:]
	if len(statements) != 1 {
		t.Fatalf("statements = %q, want one metadata lookup", statements)
	}
	if !strings.Contains(statements[0], "unnest(") {
		t.Errorf("statement = %q, want the table layout query", statements[0])
	}
}

func TestResolveKnownTableConfiguration(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	events := domain.TableIdentifier{Schema: "public", Name: "events"}
	unknown := domain.TableIdentifier{Schema: "other", Name: "logs"}

	if _, err := introspector.ResolveKnownTableConfiguration(events); !errors.Is(err, errIntrospectionRequired) {
		t.Fatalf("ResolveKnownTableConfiguration() error = %v, want errIntrospectionRequired before the first introspection", err)
	}

	if _, err := introspector.ResolveTableConfiguration(context.Background(), events); err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), unknown); !errors.Is(err, ErrTableIgnored) {
		t.Fatalf("ResolveTableConfiguration() error = %v, want ErrTableIgnored", err)
	}

	before := len(server.recorded())

	cases := []struct {
		name  string
		table domain.TableIdentifier
		want  domain.TableIdentifier
		err   error
	}{
		{name: "configured table", table: events, want: events},
		{name: "known partition", table: domain.TableIdentifier{Schema: "public", Name: "events_2024"}, want: events},
		{name: "ignored table", table: unknown, err: ErrTableIgnored},
		{name: "unseen table", table: domain.TableIdentifier{Schema: "public", Name: "events_2025"}, err: errIntrospectionRequired},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := introspector.ResolveKnownTableConfiguration(tt.table)
			if !errors.Is(err, tt.err) {
				t.Fatalf("ResolveKnownTableConfiguration() error = %v, want %v", err, tt.err)
			}
			if ref != tt.want {
				t.Errorf("ResolveKnownTableConfiguration() = %+v, want %+v", ref, tt.want)
			}
		})
	}

	if after := len(server.recorded()); after != before {
		t.Errorf("statements grew from %d to %d, want a registry-only lookup to reach no server", before, after)
	}
}

func TestInvalidateSkipsReplicaIdentityForUnmanagedTables(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	ensure, calls := replicaIdentityRecorder()
	introspector := newTestIntrospector(t, server, []config.Table{unmanagedTable("public", "users")}, ensure)

	if err := introspector.Invalidate(context.Background(), nil); err != nil {
		t.Fatalf("Invalidate() error = %v", err)
	}

	if len(*calls) != 1 {
		t.Fatalf("ensureReplicaIdentity calls = %d, want 1", len(*calls))
	}
	if targets := (*calls)[0]; len(targets) != 0 {
		t.Errorf("ensureReplicaIdentity tables = %+v, want none: hump manages no replica identity here", targets)
	}
}

func TestResolveTableConfigurationIgnoresUnknownTable(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	unknown := domain.TableIdentifier{Schema: "other", Name: "logs"}
	ref, err := introspector.ResolveTableConfiguration(context.Background(), unknown)
	if !errors.Is(err, ErrTableIgnored) {
		t.Fatalf("ResolveTableConfiguration() error = %v, want ErrTableIgnored", err)
	}
	if ref != (domain.TableIdentifier{}) {
		t.Errorf("ResolveTableConfiguration() = %+v, want the zero table", ref)
	}
	if _, ignored := introspector.ignoredTables[unknown]; !ignored {
		t.Fatal("an unresolvable table must be ignored from then on")
	}

	before := len(server.recorded())
	if _, err := introspector.ResolveTableConfiguration(context.Background(), unknown); !errors.Is(err, ErrTableIgnored) {
		t.Fatalf("second ResolveTableConfiguration() error = %v, want ErrTableIgnored", err)
	}
	if after := len(server.recorded()); after != before {
		t.Errorf("statements grew from %d to %d, want an ignored table to skip introspection", before, after)
	}
}

func TestResolveTableConfigurationPropagatesRetryError(t *testing.T) {
	server := newFakeServer(t)
	var layouts int
	server.setResponder(func(sql string) queryResult {
		if strings.Contains(sql, "pg_extension") {
			return singleValue("f")
		}

		layouts++
		if layouts > 1 {
			return serverError("42501", "permission denied")
		}

		return tableLayout(layoutRow("public", "users", tableTypeRegular, nil, nil))
	})
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	unknown := domain.TableIdentifier{Schema: "other", Name: "logs"}
	_, err := introspector.ResolveTableConfiguration(context.Background(), unknown)
	if err == nil {
		t.Fatal("ResolveTableConfiguration() error = nil, want the retry error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %v, want the retry error", err)
	}
	if _, ignored := introspector.ignoredTables[unknown]; ignored {
		t.Error("a table whose retry failed must not be ignored: the lookup never completed")
	}
}

func TestGetTablesMetadataRejectsEmptyTableList(t *testing.T) {
	server := newFakeServer(t)
	introspector := newTestIntrospector(t, server, nil, nil)

	_, err := introspector.getTablesMetadata(context.Background(), introspector.connection, nil)
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "other", Name: "events"},
	}

	if _, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables); err != nil {
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
		{name: "without timescaledb", timescale: false, want: introspectorTablesNoTimescaleSql},
		{name: "with timescaledb", timescale: true, want: introspectorTablesTimescaleSql},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(tableLayoutResponder(tt.timescale,
				layoutRow("public", "users", tableTypeRegular, nil, nil),
			))
			introspector := newTestIntrospector(t, server, nil, nil)

			tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}
			if _, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables); err != nil {
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "events"},
		{Schema: "public", Name: "metrics"},
		{Schema: "public", Name: "users"},
	}

	info, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "events"}}

	info, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "gone"},
		{Schema: "audit", Name: "trail"},
	}

	_, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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
	introspector := newTestIntrospector(t, server, nil, nil)

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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
	introspector := newTestIntrospector(t, server, nil, nil)
	introspector.newTypeMap = newUnencodableTypeMap

	tables := []domain.TableIdentifier{{Schema: "public", Name: "users"}}

	_, err := introspector.getTablesMetadata(context.Background(), introspector.connection, tables)
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

func TestIntrospectTablesIndexesPartitionsByParent(t *testing.T) {
	metadata := tablesMetadata{
		{Schema: "public", Name: "events"}: {
			Type:       tableTypePartitioned,
			Partitions: []domain.TableIdentifier{{Schema: "public", Name: "events_2024"}},
		},
		{Schema: "public", Name: "metrics"}: {
			Type:       tableTypeHypertable,
			Partitions: []domain.TableIdentifier{{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"}},
		},
		{Schema: "public", Name: "users"}: {Type: tableTypeRegular},
	}

	introspector := newTestIntrospector(t, newFakeServer(t), nil, nil)

	result := introspector.introspectTables(metadata)

	want := map[domain.TableIdentifier]domain.TableIdentifier{
		{Schema: "public", Name: "events_2024"}:                     {Schema: "public", Name: "events"},
		{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"}: {Schema: "public", Name: "metrics"},
	}
	if len(result.Partitions) != len(want) {
		t.Fatalf("Partitions = %+v, want %+v", result.Partitions, want)
	}
	for partition, parent := range want {
		if got := result.Partitions[partition]; got != parent {
			t.Errorf("Partitions[%+v] = %+v, want %+v", partition, got, parent)
		}
	}

	users := result.Tables[domain.TableIdentifier{Schema: "public", Name: "users"}]
	if users == nil || users.Type != tableTypeRegular || users.Partitions != nil {
		t.Errorf("users = %+v, want a regular table with no partitions", users)
	}

	events := result.Tables[domain.TableIdentifier{Schema: "public", Name: "events"}]
	if len(events.Partitions) != 1 {
		t.Errorf("events partitions = %+v, want 1", events.Partitions)
	}
}

func TestIntrospectTablesSkipsUnnamedPartitions(t *testing.T) {
	metrics := domain.TableIdentifier{Schema: "public", Name: "metrics"}
	metadata := tablesMetadata{
		metrics: {
			Type:       tableTypeHypertable,
			Partitions: []domain.TableIdentifier{{Schema: "_timescaledb_internal"}},
		},
	}

	introspector := newTestIntrospector(t, newFakeServer(t), nil, nil)

	result := introspector.introspectTables(metadata)

	if len(result.Partitions) != 0 {
		t.Errorf("Partitions = %+v, want none: a chunkless hypertable has no partition to index", result.Partitions)
	}
	if partitions := result.Tables[metrics].Partitions; partitions != nil {
		t.Errorf("metrics partitions = %+v, want none", partitions)
	}
}

func TestResolveTableByPartition(t *testing.T) {
	events := domain.TableIdentifier{Schema: "public", Name: "events"}
	metrics := domain.TableIdentifier{Schema: "public", Name: "metrics"}

	cases := []struct {
		name      string
		metadata  tablesMetadata
		partition domain.TableIdentifier
		want      *domain.TableIdentifier
	}{
		{
			name: "partition matched by name",
			metadata: tablesMetadata{events: {
				Type:       tableTypePartitioned,
				Partitions: []domain.TableIdentifier{{Schema: "public", Name: "events_2024"}},
			}},
			partition: domain.TableIdentifier{Schema: "public", Name: "events_2024"},
			want:      &events,
		},
		{
			name: "hypertable chunk matched by schema alone",
			metadata: tablesMetadata{metrics: {
				Type:       tableTypeHypertable,
				Partitions: []domain.TableIdentifier{{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"}},
			}},
			partition: domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_9_9_chunk"},
			want:      &metrics,
		},
		{
			name: "name mismatch in the same schema",
			metadata: tablesMetadata{events: {
				Type:       tableTypePartitioned,
				Partitions: []domain.TableIdentifier{{Schema: "public", Name: "events_2024"}},
			}},
			partition: domain.TableIdentifier{Schema: "public", Name: "events_2025"},
			want:      nil,
		},
		{
			name: "partition in another schema",
			metadata: tablesMetadata{events: {
				Type:       tableTypePartitioned,
				Partitions: []domain.TableIdentifier{{Schema: "public", Name: "events_2024"}},
			}},
			partition: domain.TableIdentifier{Schema: "archive", Name: "events_2024"},
			want:      nil,
		},
		{
			name:      "table without partitions",
			metadata:  tablesMetadata{events: {Type: tableTypeRegular}},
			partition: domain.TableIdentifier{Schema: "public", Name: "events_2024"},
			want:      nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			introspector := newTestIntrospector(t, newFakeServer(t), nil, nil)

			got := introspector.resolveTableByPartition(tt.metadata, tt.partition)

			switch {
			case tt.want == nil && got != nil:
				t.Errorf("resolveTableByPartition() = %+v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("resolveTableByPartition() = nil, want %+v", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Errorf("resolveTableByPartition() = %+v, want %+v", *got, *tt.want)
			}
		})
	}
}

func TestReplicaIdentityTargetsCoversTablesAndPartitions(t *testing.T) {
	events := domain.TableIdentifier{Schema: "public", Name: "events"}
	metrics := domain.TableIdentifier{Schema: "public", Name: "metrics"}
	users := domain.TableIdentifier{Schema: "public", Name: "users"}

	cases := []struct {
		name     string
		metadata tablesMetadata
		managed  []domain.TableIdentifier
		want     []domain.TableIdentifier
	}{
		{
			name:     "regular table",
			metadata: tablesMetadata{users: {Type: tableTypeRegular}},
			managed:  []domain.TableIdentifier{users},
			want:     []domain.TableIdentifier{users},
		},
		{
			name: "partitioned table and its partitions",
			metadata: tablesMetadata{events: {
				Type: tableTypePartitioned,
				Partitions: []domain.TableIdentifier{
					{Schema: "public", Name: "events_2024"},
					{Schema: "archive", Name: "events_2025"},
				},
			}},
			managed: []domain.TableIdentifier{events},
			want: []domain.TableIdentifier{
				events,
				{Schema: "public", Name: "events_2024"},
				{Schema: "archive", Name: "events_2025"},
			},
		},
		{
			name: "duplicate partitions are listed once",
			metadata: tablesMetadata{events: {
				Type: tableTypePartitioned,
				Partitions: []domain.TableIdentifier{
					{Schema: "public", Name: "events_2024"},
					{Schema: "public", Name: "events_2024"},
				},
			}},
			managed: []domain.TableIdentifier{events},
			want:    []domain.TableIdentifier{events, {Schema: "public", Name: "events_2024"}},
		},
		{
			name: "hypertable chunks are left alone",
			metadata: tablesMetadata{metrics: {
				Type:       tableTypeHypertable,
				Partitions: []domain.TableIdentifier{{Schema: "_timescaledb_internal", Name: "_hyper_1_1"}},
			}},
			managed: []domain.TableIdentifier{metrics},
			want:    []domain.TableIdentifier{metrics},
		},
		{
			name: "partitions missing a schema or a name are skipped",
			metadata: tablesMetadata{events: {
				Type: tableTypePartitioned,
				Partitions: []domain.TableIdentifier{
					{Name: "orphan"},
					{Schema: "public"},
				},
			}},
			managed: []domain.TableIdentifier{events},
			want:    []domain.TableIdentifier{events},
		},
		{
			name:     "unmanaged tables contribute nothing",
			metadata: tablesMetadata{users: {Type: tableTypeRegular}, events: {Type: tableTypePartitioned}},
			managed:  []domain.TableIdentifier{users},
			want:     []domain.TableIdentifier{users},
		},
		{
			name:     "no managed tables",
			metadata: tablesMetadata{users: {Type: tableTypeRegular}},
			managed:  nil,
			want:     nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			introspector := newTestIntrospector(t, newFakeServer(t), nil, nil)

			got := introspector.replicaIdentityTargets(tt.metadata, tt.managed)

			if len(got) != len(tt.want) {
				t.Fatalf("replicaIdentityTargets() = %+v, want %+v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("target %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReplicaIdentityTargetsFollowsConfiguredOrder(t *testing.T) {
	managed := []domain.TableIdentifier{
		{Schema: "public", Name: "zebras"},
		{Schema: "public", Name: "apples"},
		{Schema: "public", Name: "mangos"},
	}
	metadata := tablesMetadata{}
	for _, table := range managed {
		metadata[table] = &tableMetadata{Type: tableTypeRegular}
	}

	introspector := newTestIntrospector(t, newFakeServer(t), nil, nil)

	for range 10 {
		got := introspector.replicaIdentityTargets(metadata, managed)

		for i := range managed {
			if got[i] != managed[i] {
				t.Fatalf("target %d = %+v, want %+v: the order must not depend on map iteration", i, got[i], managed[i])
			}
		}
	}
}

func TestIntrospectorHasTimescaleDB(t *testing.T) {
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
			introspector := newTestIntrospector(t, server, nil, nil)

			got, err := introspector.hasTimescaleDB(context.Background(), introspector.connection)
			if err != nil {
				t.Fatalf("hasTimescaleDB() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("hasTimescaleDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntrospectorHasTimescaleDBProbesOnce(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return singleValue("t") })
	introspector := newTestIntrospector(t, server, nil, nil)

	for range 3 {
		got, err := introspector.hasTimescaleDB(context.Background(), introspector.connection)
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

func TestIntrospectorHasTimescaleDBRejectsEmptyResult(t *testing.T) {
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
			introspector := newTestIntrospector(t, server, nil, nil)

			_, err := introspector.hasTimescaleDB(context.Background(), introspector.connection)
			if !errors.Is(err, database.ErrDbSetup) {
				t.Fatalf("hasTimescaleDB() error = %v, want ErrDbSetup", err)
			}
		})
	}
}

func TestIntrospectorHasTimescaleDBPropagatesServerError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })
	introspector := newTestIntrospector(t, server, nil, nil)

	if _, err := introspector.hasTimescaleDB(context.Background(), introspector.connection); err == nil {
		t.Fatal("hasTimescaleDB() error = nil, want the server error")
	}
}

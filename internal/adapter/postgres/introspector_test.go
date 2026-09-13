package postgres

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"hump/config"
	"hump/internal/domain"
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
	server.setResponder(withParent(stagedLayoutResponder(
		[][][]byte{layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024"))},
		[][][]byte{
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2025")),
		},
	), domain.TableIdentifier{Schema: "public", Name: "events"}))
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

func TestResolveTableConfigurationCostsTwoQueriesPerOwnedMiss(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(withParent(stagedLayoutResponder(
		[][][]byte{layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024"))},
		[][][]byte{
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")),
			layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2025")),
		},
	), domain.TableIdentifier{Schema: "public", Name: "events"}))
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
	if len(statements) != 2 {
		t.Fatalf("statements = %q, want the parent lookup and the metadata lookup", statements)
	}
	if !strings.Contains(statements[0], "inheritance parent") {
		t.Errorf("statement = %q, want the parent lookup first", statements[0])
	}
	if !strings.Contains(statements[1], "unnest(") {
		t.Errorf("statement = %q, want the table layout query second", statements[1])
	}
}

func TestResolveTableConfigurationCostsOneQueryPerForeignMiss(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(tableLayoutResponder(true,
		layoutRow("public", "metrics", tableTypeHypertable, []byte("_timescaledb_internal"), []byte("_hyper_1_1_chunk")),
	))
	ensure, calls := replicaIdentityRecorder()
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "metrics")}, ensure)

	known := domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), known); err != nil {
		t.Fatalf("ResolveTableConfiguration() error = %v", err)
	}

	before := len(server.recorded())
	beforeEnsures := len(*calls)

	foreign := domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_9_9_chunk"}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), foreign); !errors.Is(err, ErrTableIgnored) {
		t.Fatalf("ResolveTableConfiguration() error = %v, want ErrTableIgnored", err)
	}

	statements := server.recorded()[before:]
	if len(statements) != 1 {
		t.Fatalf("statements = %q, want only the parent lookup", statements)
	}
	if !strings.Contains(statements[0], "inheritance parent") {
		t.Errorf("statement = %q, want the parent lookup", statements[0])
	}
	if after := len(*calls); after != beforeEnsures {
		t.Errorf("replica identity passes = %d, want %d: a foreign chunk must not trigger DDL", after, beforeEnsures)
	}
}

func TestResolveTableConfigurationIgnoresChunkOfUnconfiguredHypertable(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(withParent(tableLayoutResponder(true,
		layoutRow("public", "metrics", tableTypeHypertable, []byte("_timescaledb_internal"), []byte("_hyper_1_1_chunk")),
	), domain.TableIdentifier{Schema: "public", Name: "other_metrics"}))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "metrics")}, nil)

	foreign := domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_9_9_chunk"}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), foreign); !errors.Is(err, ErrTableIgnored) {
		t.Fatalf("ResolveTableConfiguration() error = %v, want ErrTableIgnored", err)
	}
	if _, ignored := introspector.ignoredTables[foreign]; !ignored {
		t.Error("a chunk of an unconfigured hypertable must be ignored from then on")
	}
}

func TestResolveTableConfigurationIgnoresInheritanceChildOfRegularTable(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(withParent(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
	), domain.TableIdentifier{Schema: "public", Name: "users"}))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	child := domain.TableIdentifier{Schema: "public", Name: "users_archive"}
	if _, err := introspector.ResolveTableConfiguration(context.Background(), child); !errors.Is(err, ErrTableIgnored) {
		t.Fatalf("ResolveTableConfiguration() error = %v, want ErrTableIgnored", err)
	}
}

func TestResolveTableConfigurationPropagatesParentLookupError(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(failingOn(
		tableLayoutResponder(false, layoutRow("public", "users", tableTypeRegular, nil, nil)),
		"inheritance parent",
		serverError("42501", "permission denied"),
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	unknown := domain.TableIdentifier{Schema: "other", Name: "logs"}
	_, err := introspector.ResolveTableConfiguration(context.Background(), unknown)
	if err == nil {
		t.Fatal("ResolveTableConfiguration() error = nil, want the parent lookup error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %v, want the parent lookup error", err)
	}
	if _, ignored := introspector.ignoredTables[unknown]; ignored {
		t.Error("a table whose parent lookup failed must not be ignored: the lookup never completed")
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
	// The parent lookup must succeed and name a configured table, or the retry
	// this test is about never runs: an unowned relation is ignored before it.
	server.setResponder(withParent(func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "pg_extension"):
			return singleValue("f")
		case strings.Contains(sql, "unnest("):
			layouts++
			if layouts > 1 {
				return serverError("42501", "permission denied")
			}

			return tableLayout(layoutRow("public", "events", tableTypePartitioned, []byte("public"), []byte("events_2024")))
		default:
			return queryResult{}
		}
	}, domain.TableIdentifier{Schema: "public", Name: "events"}))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "events")}, nil)

	partition := domain.TableIdentifier{Schema: "public", Name: "events_2025"}
	_, err := introspector.ResolveTableConfiguration(context.Background(), partition)
	if err == nil {
		t.Fatal("ResolveTableConfiguration() error = nil, want the retry error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %v, want the retry error", err)
	}
	if _, ignored := introspector.ignoredTables[partition]; ignored {
		t.Error("a table whose retry failed must not be ignored: the lookup never completed")
	}
}

func TestGetTableParentRejectsShortRows(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(failingOn(
		tableLayoutResponder(false, layoutRow("public", "users", tableTypeRegular, nil, nil)),
		"inheritance parent",
		queryResult{columns: 1, rows: [][][]byte{{[]byte("public")}}},
	))
	introspector := newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, nil)

	unknown := domain.TableIdentifier{Schema: "other", Name: "logs"}
	_, err := introspector.ResolveTableConfiguration(context.Background(), unknown)
	if err == nil {
		t.Fatal("ResolveTableConfiguration() error = nil, want the column count error")
	}
	if !strings.Contains(err.Error(), "get parent of other.logs: expected 2 columns, got 1") {
		t.Errorf("error = %v, want it to name the table and the column count", err)
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
			name: "hypertable chunk matched by name",
			metadata: tablesMetadata{metrics: {
				Type:       tableTypeHypertable,
				Partitions: []domain.TableIdentifier{{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"}},
			}},
			partition: domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"},
			want:      &metrics,
		},
		{
			name: "chunk of another hypertable in the same schema",
			metadata: tablesMetadata{metrics: {
				Type:       tableTypeHypertable,
				Partitions: []domain.TableIdentifier{{Schema: "_timescaledb_internal", Name: "_hyper_1_1_chunk"}},
			}},
			partition: domain.TableIdentifier{Schema: "_timescaledb_internal", Name: "_hyper_9_9_chunk"},
			want:      nil,
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

func TestReplicaIdentityTargetsSkipsTablesMissingFromMetadata(t *testing.T) {
	present := domain.TableIdentifier{Schema: "public", Name: "users"}
	dropped := domain.TableIdentifier{Schema: "public", Name: "logs"}

	metadata := tablesMetadata{present: &tableMetadata{Type: tableTypeRegular}}
	introspector := newTestIntrospector(t, newFakeServer(t), nil, nil)

	got := introspector.replicaIdentityTargets(metadata, []domain.TableIdentifier{dropped, present})

	want := []domain.TableIdentifier{present}
	if !slices.Equal(got, want) {
		t.Errorf("targets = %+v, want %+v: a table absent from the snapshot has nothing to alter", got, want)
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

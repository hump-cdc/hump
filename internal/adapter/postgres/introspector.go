package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"

	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/introspector_tables_timescale.sql
var introspectorTablesTimescaleSql string

//go:embed sql/introspector_tables_no_timescale.sql
var introspectorTablesNoTimescaleSql string

//go:embed sql/introspector_has_timescale.sql
var introspectorHasTimescaleSql string

// introspector manages in-memory table configuration in response to schema
// changes (e.g., new partitions) and ensures replica identity is set for
// replicated tables.
type introspector struct {
	connection  *connection
	registry    *introspectorRegistry
	mu          sync.Mutex
	initialized bool // initialized reports whether the first full introspection has run.
	// ignoredTables holds tables a re-introspection still failed to resolve;
	// later lookups for them skip introspection entirely. Entries are never
	// removed, so a table stays ignored for the lifetime of the introspector.
	ignoredTables         map[domain.TableIdentifier]struct{}
	ensureReplicaIdentity func(ctx context.Context, conn *connection, tables []domain.TableIdentifier) error
	// newTypeMap builds the pgtype map that encodes array bind parameters. A
	// fresh map per call keeps its encode-plan cache free of concurrent writes;
	// tests replace it with one that refuses to encode.
	newTypeMap func() *pgtype.Map
}

// newIntrospector returns an introspector backed by connection and registry, calling
// ensureReplicaIdentity after every successful introspection. conn is borrowed:
// the caller must keep it open for the lifetime of the introspector.
func newIntrospector(
	conn *connection,
	registry *introspectorRegistry,
	ensureReplicaIdentity func(ctx context.Context, conn *connection, tables []domain.TableIdentifier) error,
) *introspector {
	return &introspector{
		connection:            conn,
		registry:              registry,
		ignoredTables:         make(map[domain.TableIdentifier]struct{}),
		ensureReplicaIdentity: ensureReplicaIdentity,
		newTypeMap:            pgtype.NewMap,
	}
}

// Invalidate re-introspects all configured tables and refreshes the registry's
// partition→parent mapping. A non-nil table narrows the work: if it turns out
// not to be a partition, the call is a no-op.
func (i *introspector) Invalidate(ctx context.Context, table *domain.TableIdentifier) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.invalidate(ctx, i.connection, table)
}

// ResolveTableConfiguration returns the resolved table — the parent when table
// is a partition — and its configured sinks. A miss costs one re-introspection;
// if that still misses, the table is ignored from then on and every lookup for
// it returns ErrTableIgnored.
func (i *introspector) ResolveTableConfiguration(ctx context.Context, table domain.TableIdentifier) (domain.TableIdentifier, []sinkcfg.Config, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// populate the registry on first call
	if !i.initialized {
		if err := i.invalidate(ctx, i.connection, nil); err != nil {
			return domain.TableIdentifier{}, nil, err
		}
	}

	if ref, sinks := i.registry.Resolve(table); sinks != nil {
		return ref, sinks, nil
	}

	// a previous re-introspection already failed to find it
	if _, ignored := i.ignoredTables[table]; ignored {
		return domain.TableIdentifier{}, nil, ErrTableIgnored
	}

	// retry once, in case table is a partition created after startup
	if err := i.invalidate(ctx, i.connection, &table); err != nil {
		return domain.TableIdentifier{}, nil, err
	}

	ref, sinks := i.registry.Resolve(table)
	if sinks == nil {
		i.ignoredTables[table] = struct{}{}
		return domain.TableIdentifier{}, nil, ErrTableIgnored
	}

	return ref, sinks, nil
}

// invalidate performs the introspection refresh while the caller holds
// i.mu. See Invalidate for the semantics of the table argument.
func (i *introspector) invalidate(ctx context.Context, conn *connection, table *domain.TableIdentifier) error {
	// a non-nil table means an earlier lookup missed it; only refresh the
	// registry if it proves to be a partition
	if table != nil {
		partition, err := i.resolveTableByPartition(ctx, conn, *table, i.registry.List())
		if err != nil {
			return err
		}

		// not a partition of any configured table, so a refresh would not resolve it
		if partition == nil {
			return nil
		}
	}

	introspection, err := i.introspectTables(ctx, conn, i.registry.List())
	if err != nil {
		return err
	}

	i.registry.UpdatePartitions(introspection.Partitions)

	if err := i.ensureReplicaIdentity(ctx, conn, i.registry.ReplicaIdentityFullTables()); err != nil {
		return err
	}

	if table == nil {
		i.initialized = true
	}

	return nil
}

// getTablesMetadata queries the database and returns the configured tables with
// their partition layout. Returns database.ErrNoTablesConfigured when tables is
// empty, database.ErrTablesNotFound when any configured table is missing (the
// error lists every one).
func (i *introspector) getTablesMetadata(ctx context.Context, conn *connection, tables []domain.TableIdentifier) (tablesMetadata, error) {
	if len(tables) == 0 {
		return nil, database.ErrNoTablesConfigured
	}

	timescale, err := i.hasTimescaleDB(ctx, conn)
	if err != nil {
		return nil, err
	}

	schemas := make([]string, len(tables))
	names := make([]string, len(tables))
	for idx, t := range tables {
		schemas[idx] = string(t.Schema)
		names[idx] = string(t.Name)
	}

	sql := introspectorTablesNoTimescaleSql
	if timescale {
		sql = introspectorTablesTimescaleSql
	}

	arrays := []struct {
		name   string
		values []string
	}{
		{name: "schemas", values: schemas},
		{name: "names", values: names},
	}

	m := i.newTypeMap()
	params := make([]param, 0, len(arrays))
	for _, array := range arrays {
		encoded, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](array.values), nil)
		if err != nil {
			return nil, fmt.Errorf("encode %s array: %w", array.name, err)
		}

		params = append(params, param{Value: encoded, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode})
	}

	queryResult, err := conn.ExecParams(ctx, sql, params, nil)
	if err != nil {
		return nil, fmt.Errorf("get tables with partitions: %w", err)
	}

	result := make(tablesMetadata)
	for _, values := range queryResult.Rows {
		if len(values) < 5 {
			return nil, fmt.Errorf("get tables with partitions: expected 5 columns, got %d", len(values))
		}

		table := domain.TableIdentifier{Schema: domain.TableSchema(values[0]), Name: domain.TableName(values[1])}
		kind := parseTableType(string(values[2]))
		partitionSchema := values[3]
		partitionTable := values[4]

		info, ok := result[table]
		if !ok {
			info = &tableMetadata{Type: kind}
			result[table] = info
		}

		if partitionSchema != nil {
			p := domain.TableIdentifier{Schema: domain.TableSchema(partitionSchema)}
			if partitionTable != nil {
				p.Name = domain.TableName(partitionTable)
			}
			info.Partitions = append(info.Partitions, p)
		}
	}

	var missing []string
	for _, t := range tables {
		if _, ok := result[t]; !ok {
			missing = append(missing, string(t.Schema)+"."+string(t.Name))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", database.ErrTablesNotFound, strings.Join(missing, ", "))
	}

	return result, nil
}

// introspectTables runs tablesWithPartitions and reshapes the result into a
// parent map plus a flat partition→parent reverse index.
func (i *introspector) introspectTables(ctx context.Context, conn *connection, tables []domain.TableIdentifier) (*tablesIntrospection, error) {
	dbTables, err := i.getTablesMetadata(ctx, conn, tables)
	if err != nil {
		return nil, err
	}

	result := &tablesIntrospection{
		Tables:     make(map[domain.TableIdentifier]*introspectedTable),
		Partitions: make(map[domain.TableIdentifier]domain.TableIdentifier),
	}

	for parent, info := range dbTables {
		entry := &introspectedTable{Type: info.Type}
		result.Tables[parent] = entry

		if info.Type != tableTypePartitioned && info.Type != tableTypeHypertable {
			continue
		}

		for _, p := range info.Partitions {
			if p.Name == "" {
				continue
			}

			entry.Partitions = append(entry.Partitions, p)
			result.Partitions[p] = parent
		}
	}

	return result, nil
}

// resolveTableByPartition returns the parent whose partition list contains partition,
// or nil when no configured table claims it.
func (i *introspector) resolveTableByPartition(ctx context.Context, conn *connection, partition domain.TableIdentifier, tables []domain.TableIdentifier) (*domain.TableIdentifier, error) {
	metadata, err := i.getTablesMetadata(ctx, conn, tables)
	if err != nil {
		return nil, err
	}

	for parent, info := range metadata {
		for _, p := range info.Partitions {
			if p.Schema != partition.Schema {
				continue
			}
			if info.Type == tableTypeHypertable || p.Name == partition.Name {
				return &parent, nil
			}
		}
	}

	return nil, nil
}

// hasTimescaleDB reports whether the timescaledb extension is installed, which
// decides which table layout query applies. An empty result wraps
// database.ErrDbSetup; query errors pass through.
func (i *introspector) hasTimescaleDB(ctx context.Context, conn *connection) (bool, error) {
	results, err := conn.Exec(ctx, introspectorHasTimescaleSql)
	if err != nil {
		return false, err
	}
	if len(results) == 0 || len(results[0].Rows) == 0 {
		return false, fmt.Errorf("empty timescaledb extension result: %w", database.ErrDbSetup)
	}

	return string(results[0].Rows[0][0]) == "t", nil
}

// introspectedTable holds the resolved table kind together with the partitions
// that belong to it. Partitions is empty for non-partitioned tables.
type introspectedTable struct {
	Type       tableType
	Partitions []domain.TableIdentifier
}

// tablesIntrospection is the reshaped result of introspectTables: a parent
// table view (Tables) and a flat partition→parent reverse index (Partitions).
type tablesIntrospection struct {
	// Tables maps a configured table to its info with partitions.
	Tables map[domain.TableIdentifier]*introspectedTable
	// Partitions maps a partition to its parent table.
	Partitions map[domain.TableIdentifier]domain.TableIdentifier
}

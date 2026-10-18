package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sync"

	"hump/internal/domain"
)

//go:embed sql/introspector_table_parent.sql
var introspectorTableParentSql string

// introspector manages in-memory table configuration in response to schema
// changes (e.g., new partitions) and ensures replica identity is set for
// replicated tables.
type introspector struct {
	*tableMetadataLoader
	registry    *introspectorRegistry
	mu          sync.Mutex
	initialized bool // initialized reports whether the first full introspection has run.
	// ignoredTables holds tables a re-introspection still failed to resolve;
	// later lookups for them skip introspection entirely. Entries are never
	// removed, so a table stays ignored for the lifetime of the introspector.
	ignoredTables         map[domain.TableIdentifier]struct{}
	ensureReplicaIdentity func(ctx context.Context, conn *connection, tables []domain.TableIdentifier) error
}

// newIntrospector returns an introspector backed by connection and registry,
// calling ensureReplicaIdentity with the expanded target tables after every
// successful introspection. conn is borrowed: the caller must keep it open for
// the lifetime of the introspector.
func newIntrospector(
	conn *connection,
	registry *introspectorRegistry,
	ensureReplicaIdentity func(ctx context.Context, conn *connection, tables []domain.TableIdentifier) error,
) *introspector {
	return &introspector{
		tableMetadataLoader:   newTableMetadataLoader(conn),
		registry:              registry,
		ignoredTables:         make(map[domain.TableIdentifier]struct{}),
		ensureReplicaIdentity: ensureReplicaIdentity,
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
// is a partition. A miss costs a parent lookup, plus one re-introspection when a
// configured table owns it; anything left unresolved is ignored from then on and
// every lookup for it returns ErrTableIgnored.
func (i *introspector) ResolveTableConfiguration(ctx context.Context, table domain.TableIdentifier) (domain.TableIdentifier, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// populate the registry on first call
	if !i.initialized {
		if err := i.invalidate(ctx, i.connection, nil); err != nil {
			return domain.TableIdentifier{}, err
		}
	}

	if ref, err := i.resolveKnown(table); !errors.Is(err, errIntrospectionRequired) {
		return ref, err
	}

	// A relation the registry misses is only worth a refresh when a configured
	// table owns it. Resolving its parent first keeps every other relation —
	// chunks of hypertables hump does not replicate, which the publication pulls
	// in schema-wide — off the expensive path.
	parent, ok, err := i.getTableParent(ctx, i.connection, table)
	if err != nil {
		return domain.TableIdentifier{}, err
	}
	if !ok || !i.registry.IsConfigured(parent) {
		i.ignoredTables[table] = struct{}{}
		return domain.TableIdentifier{}, ErrTableIgnored
	}

	// retry once, in case table is a partition created after startup
	if err := i.invalidate(ctx, i.connection, &table); err != nil {
		return domain.TableIdentifier{}, err
	}

	ref, ok := i.registry.Resolve(table)
	if !ok {
		i.ignoredTables[table] = struct{}{}
		return domain.TableIdentifier{}, ErrTableIgnored
	}

	return ref, nil
}

// ResolveKnownTableConfiguration answers from the in-memory registry alone,
// never touching the database. It returns errIntrospectionRequired when only a
// catalog refresh can decide, which is what ResolveTableConfiguration is for.
func (i *introspector) ResolveKnownTableConfiguration(table domain.TableIdentifier) (domain.TableIdentifier, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.resolveKnown(table)
}

// resolveKnown performs the registry-only lookup while the caller holds i.mu.
// See ResolveKnownTableConfiguration for the semantics.
func (i *introspector) resolveKnown(table domain.TableIdentifier) (domain.TableIdentifier, error) {
	if !i.initialized {
		return domain.TableIdentifier{}, errIntrospectionRequired
	}

	if ref, ok := i.registry.Resolve(table); ok {
		return ref, nil
	}

	// a previous re-introspection already failed to find it
	if _, ignored := i.ignoredTables[table]; ignored {
		return domain.TableIdentifier{}, ErrTableIgnored
	}

	return domain.TableIdentifier{}, errIntrospectionRequired
}

// invalidate performs the introspection refresh while the caller holds
// i.mu. See Invalidate for the semantics of the table argument.
func (i *introspector) invalidate(ctx context.Context, conn *connection, table *domain.TableIdentifier) error {
	metadata, err := i.getTablesMetadata(ctx, i.registry.List())
	if err != nil {
		return err
	}

	// a non-nil table means an earlier lookup missed it; one that is not a
	// partition of a configured table would not be resolved by a refresh either
	if table != nil && i.resolveTableByPartition(metadata, *table) == nil {
		return nil
	}

	i.registry.UpdatePartitions(i.introspectTables(metadata).Partitions)

	targets := i.replicaIdentityTargets(metadata, i.registry.ReplicaIdentityFullTables())
	if err := i.ensureReplicaIdentity(ctx, conn, targets); err != nil {
		return err
	}

	if table == nil {
		i.initialized = true
	}

	return nil
}

// getTableParent returns the inheritance parent of table — the hypertable owning
// a chunk, or the partitioned table owning a partition — and false when table is
// not an inheritance child. Only the immediate parent is reported.
func (i *introspector) getTableParent(ctx context.Context, conn *connection, table domain.TableIdentifier) (domain.TableIdentifier, bool, error) {
	queryResult, err := conn.ExecParams(ctx, introspectorTableParentSql, tableScalarParams(table), nil)
	if err != nil {
		return domain.TableIdentifier{}, false, fmt.Errorf("get parent of %s.%s: %w", table.Schema, table.Name, err)
	}

	if len(queryResult.Rows) == 0 {
		return domain.TableIdentifier{}, false, nil
	}

	values := queryResult.Rows[0]
	if len(values) < 2 {
		return domain.TableIdentifier{}, false, fmt.Errorf("get parent of %s.%s: expected 2 columns, got %d", table.Schema, table.Name, len(values))
	}

	parent := domain.TableIdentifier{
		Schema: domain.TableSchema(values[0]),
		Name:   domain.TableName(values[1]),
	}

	return parent, true, nil
}

// introspectTables reshapes a catalog snapshot into a parent table view plus a
// flat partition→parent reverse index.
func (i *introspector) introspectTables(dbTables tablesMetadata) *tablesIntrospection {
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

	return result
}

// resolveTableByPartition returns the parent whose partition list names
// partition, or nil when no table in metadata claims it.
func (i *introspector) resolveTableByPartition(metadata tablesMetadata, partition domain.TableIdentifier) *domain.TableIdentifier {
	for parent, info := range metadata {
		for _, p := range info.Partitions {
			if p.Schema != partition.Schema {
				continue
			}
			if p.Name == partition.Name {
				return &parent
			}
		}
	}

	return nil
}

// replicaIdentityTargets expands managed into every table whose replica identity
// hump sets: the table itself plus its partitions. TimescaleDB chunks are left
// out, since they inherit from the hypertable and cannot be altered directly.
func (i *introspector) replicaIdentityTargets(metadata tablesMetadata, managed []domain.TableIdentifier) []domain.TableIdentifier {
	seen := make(map[domain.TableIdentifier]struct{})
	var targets []domain.TableIdentifier
	add := func(table domain.TableIdentifier) {
		if _, ok := seen[table]; ok {
			return
		}
		seen[table] = struct{}{}
		targets = append(targets, table)
	}

	// Driven by managed, not by metadata: ranging over the map would shuffle the
	// resulting statements between runs.
	for _, table := range managed {
		info, ok := metadata[table]
		if !ok {
			continue
		}

		add(table)
		if info.Type == tableTypeHypertable {
			continue
		}

		for _, p := range info.Partitions {
			if p.Schema == "" || p.Name == "" {
				continue
			}
			add(p)
		}
	}

	return targets
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

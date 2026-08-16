package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/bootstrapper_publication_ensure_exists.sql
var bootstrapperPublicationEnsureExistsSql string

//go:embed sql/bootstrapper_publication_ensure_removed.sql
var bootstrapperPublicationEnsureRemovedSql string

//go:embed sql/bootstrapper_replication_slot_ensure_exists.sql
var bootstrapperReplicationSlotEnsureExistsSql string

//go:embed sql/bootstrapper_replication_slot_ensure_removed.sql
var bootstrapperReplicationSlotEnsureRemovedSql string

//go:embed sql/bootstrapper_replica_identity_stale.sql
var bootstrapperReplicaIdentityStaleSql string

//go:embed sql/bootstrapper_replica_identity_set_full.sql
var bootstrapperReplicaIdentitySetFullSql string

//go:embed sql/bootstrapper_metadata_tables_timescale.sql
var bootstrapperMetadataTablesTimescaleSql string

//go:embed sql/bootstrapper_metadata_tables_no_timescale.sql
var bootstrapperMetadataTablesNoTimescaleSql string

//go:embed sql/bootstrapper_metadata_has_timescale.sql
var bootstrapperMetadataHasTimescaleSql string

//go:embed sql/bootstrapper_publication_missing_identity.sql
var bootstrapperPublicationMissingIdentitySql string

// missingIdentityLogLimit caps how many table names the replica identity warning
// spells out before it falls back to a count.
const missingIdentityLogLimit = 20

// bootstrapper ensures the server-side replication state — publication,
// replication slot and replica identity — matches the configured tables.
type bootstrapper struct {
	connection *connection
	newTypeMap func() *pgtype.Map
}

// newBootstrapper returns a bootstrapper backed by conn. conn is borrowed, not
// owned: the caller must keep it open for the lifetime of the bootstrapper.
func newBootstrapper(conn *connection) *bootstrapper {
	return &bootstrapper{connection: conn, newTypeMap: pgtype.NewMap}
}

// EnsurePublicationExists creates the publication with the configured tables,
// or updates an existing one to match them, always with
// publish_via_partition_root enabled.
func (b *bootstrapper) EnsurePublicationExists(ctx context.Context, publicationName string, tables []domain.TableIdentifier) error {
	info, err := b.getTablesWithPartitions(ctx, tables)
	if err != nil {
		return err
	}

	// Regular and declaratively-partitioned tables are listed by qualified name;
	// hypertable chunks are covered by TABLES IN SCHEMA clauses because their
	// schemas are TimescaleDB-managed.
	tableSet := make(map[string]struct{})
	schemaSet := make(map[string]struct{})

	for table, t := range info {
		qualified := b.quoteIdentifier(string(table.Schema)) + "." + b.quoteIdentifier(string(table.Name))
		switch t.Type {
		case tableTypeRegular, tableTypePartitioned:
			tableSet[qualified] = struct{}{}
		case tableTypeHypertable:
			tableSet[qualified] = struct{}{}
			for _, p := range t.Partitions {
				if p.Schema == "" {
					continue
				}
				schemaSet[string(p.Schema)] = struct{}{}
			}
		}
	}

	// Sorted so the generated SQL is identical across invocations.
	var parts []string
	if tableList := slices.Sorted(maps.Keys(tableSet)); len(tableList) > 0 {
		parts = append(parts, "TABLE "+strings.Join(tableList, ", "))
	}
	for _, s := range slices.Sorted(maps.Keys(schemaSet)) {
		parts = append(parts, "TABLES IN SCHEMA "+b.quoteIdentifier(s))
	}

	pubIdent := b.quoteIdentifier(publicationName)
	pubLit := b.quoteLiteral(publicationName)

	sql := fmt.Sprintf(bootstrapperPublicationEnsureExistsSql, pubLit, pubIdent, strings.Join(parts, ", "))

	if _, err := b.connection.Exec(ctx, sql); err != nil {
		return fmt.Errorf("ensure publication %q: %w", publicationName, err)
	}

	return nil
}

// WarnMissingReplicaIdentity logs the published tables whose replica identity
// cannot carry an UPDATE or a DELETE, which the server rejects for as long as
// the publication lists them. A TimescaleDB chunk schema is published whole, so
// the list reaches past the configured tables. Diagnostics only: a failed lookup
// is logged and swallowed.
func (b *bootstrapper) WarnMissingReplicaIdentity(ctx context.Context, publicationName string) {
	tables, err := b.getPublishedTablesMissingReplicaIdentity(ctx, publicationName)
	if err != nil {
		logger.Warn("Could not check replica identity of published tables", "publication", publicationName, "error", err)
		return
	}
	if len(tables) == 0 {
		return
	}

	logger.Warn("Published tables reject UPDATE and DELETE: no primary key and no REPLICA IDENTITY FULL",
		"publication", publicationName,
		"count", len(tables),
		"tables", b.summarizeTables(tables, missingIdentityLogLimit),
		"hint", "give each one a primary key or REPLICA IDENTITY FULL",
	)
}

// getPublishedTablesMissingReplicaIdentity returns the tables in the publication
// whose replica identity supports neither UPDATE nor DELETE. It reads
// pg_publication_tables, so tables published through a schema are covered too.
func (b *bootstrapper) getPublishedTablesMissingReplicaIdentity(ctx context.Context, publicationName string) ([]domain.TableIdentifier, error) {
	params := []param{textParam(publicationName)}

	queryResult, err := b.connection.ExecParams(ctx, bootstrapperPublicationMissingIdentitySql, params, nil)
	if err != nil {
		return nil, fmt.Errorf("get published tables missing replica identity: %w", err)
	}

	tables := make([]domain.TableIdentifier, 0, len(queryResult.Rows))
	for _, values := range queryResult.Rows {
		if len(values) < 2 {
			return nil, fmt.Errorf("get published tables missing replica identity: expected 2 columns, got %d", len(values))
		}

		tables = append(tables, domain.TableIdentifier{
			Schema: domain.TableSchema(values[0]),
			Name:   domain.TableName(values[1]),
		})
	}

	return tables, nil
}

// EnsurePublicationRemoved drops the publication if it exists. It is a no-op
// when the publication is absent.
func (b *bootstrapper) EnsurePublicationRemoved(ctx context.Context, publicationName string) error {
	sql := fmt.Sprintf(bootstrapperPublicationEnsureRemovedSql, b.quoteIdentifier(publicationName))

	if _, err := b.connection.Exec(ctx, sql); err != nil {
		return fmt.Errorf("remove publication %q: %w", publicationName, err)
	}

	return nil
}

// EnsureReplicationSlotExists creates the logical replication slot named
// slotName with the pgoutput plugin. A slot that exists with a different plugin
// is dropped (only possible while inactive) and recreated.
func (b *bootstrapper) EnsureReplicationSlotExists(ctx context.Context, slotName string) error {
	sql := fmt.Sprintf(bootstrapperReplicationSlotEnsureExistsSql, b.quoteLiteral(slotName))

	if _, err := b.connection.Exec(ctx, sql); err != nil {
		return fmt.Errorf("ensure replication slot %q: %w", slotName, err)
	}

	return nil
}

// EnsureReplicationSlotRemoved drops the logical replication slot named slotName
// if it exists and is inactive. It is a no-op when the slot is absent and errors
// when the slot is active, since pg_drop_replication_slot cannot drop one.
func (b *bootstrapper) EnsureReplicationSlotRemoved(ctx context.Context, slotName string) error {
	sql := fmt.Sprintf(bootstrapperReplicationSlotEnsureRemovedSql, b.quoteLiteral(slotName))

	if _, err := b.connection.Exec(ctx, sql); err != nil {
		return fmt.Errorf("remove replication slot %q: %w", slotName, err)
	}

	return nil
}

// EnsureReplicaIdentityFull sets REPLICA IDENTITY FULL on the tables that are not
// already at it. tables is the expanded target set — configured tables together
// with their partitions, TimescaleDB chunks excluded, as resolved by the caller
// from one catalog snapshot.
func (b *bootstrapper) EnsureReplicaIdentityFull(ctx context.Context, tables []domain.TableIdentifier) error {
	stale, err := b.getStaleReplicaIdentityTables(ctx, tables)
	if err != nil {
		return err
	}

	// One statement per table rather than one batched Exec: the simple protocol
	// would run a batch as a single implicit transaction, holding every
	// AccessExclusiveLock it takes until the last ALTER commits.
	for _, t := range stale {
		qualifiedIdentifier := b.quoteIdentifier(string(t.Schema)) + "." + b.quoteIdentifier(string(t.Name))

		sql := fmt.Sprintf(bootstrapperReplicaIdentitySetFullSql, qualifiedIdentifier)

		if _, err := b.connection.Exec(ctx, sql); err != nil {
			return fmt.Errorf("set replica identity full on %s.%s: %w", t.Schema, t.Name, err)
		}

		logger.Info("Table was updated to REPLICA IDENTITY FULL", "schema", t.Schema, "table", t.Name)
	}

	return nil
}

// getStaleReplicaIdentityTables returns the subset of tables whose replica
// identity is not FULL. Tables absent from the catalog are not reported: a
// partition dropped since introspection has nothing left to alter.
func (b *bootstrapper) getStaleReplicaIdentityTables(ctx context.Context, tables []domain.TableIdentifier) ([]domain.TableIdentifier, error) {
	if len(tables) == 0 {
		return nil, nil
	}

	params, err := tableArrayParams(b.newTypeMap(), tables)
	if err != nil {
		return nil, err
	}

	queryResult, err := b.connection.ExecParams(ctx, bootstrapperReplicaIdentityStaleSql, params, nil)
	if err != nil {
		return nil, fmt.Errorf("get stale replica identity tables: %w", err)
	}

	stale := make([]domain.TableIdentifier, 0, len(queryResult.Rows))
	for _, values := range queryResult.Rows {
		if len(values) < 2 {
			return nil, fmt.Errorf("get stale replica identity tables: expected 2 columns, got %d", len(values))
		}

		stale = append(stale, domain.TableIdentifier{
			Schema: domain.TableSchema(values[0]),
			Name:   domain.TableName(values[1]),
		})
	}

	return stale, nil
}

// getTablesWithPartitions returns the configured tables along with their
// partition layout (for partitioned/hypertables).
func (b *bootstrapper) getTablesWithPartitions(ctx context.Context, tables []domain.TableIdentifier) (tablesMetadata, error) {
	if len(tables) == 0 {
		return nil, database.ErrNoTablesConfigured
	}

	timescale, err := b.hasTimescaleDB(ctx)
	if err != nil {
		return nil, err
	}

	sql := bootstrapperMetadataTablesNoTimescaleSql
	if timescale {
		sql = bootstrapperMetadataTablesTimescaleSql
	}

	params, err := tableArrayParams(b.newTypeMap(), tables)
	if err != nil {
		return nil, err
	}

	queryResult, err := b.connection.ExecParams(ctx, sql, params, nil)
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

// hasTimescaleDB reports whether the timescaledb extension is installed.
func (b *bootstrapper) hasTimescaleDB(ctx context.Context) (bool, error) {
	results, err := b.connection.Exec(ctx, bootstrapperMetadataHasTimescaleSql)
	if err != nil {
		return false, err
	}
	if len(results) == 0 || len(results[0].Rows) == 0 {
		return false, fmt.Errorf("empty timescaledb extension result: %w", database.ErrDbSetup)
	}

	return string(results[0].Rows[0][0]) == "t", nil
}

// quoteLiteral wraps s as a SQL string literal, doubling embedded single quotes.
// Assumes standard_conforming_strings is on (PostgreSQL default since 9.1).
func (b *bootstrapper) quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// quoteIdentifier wraps s as a SQL identifier, doubling embedded double quotes.
func (b *bootstrapper) quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// summarizeTables renders tables as qualified names, keeping at most limit of
// them and reporting the remainder as a count so a large schema cannot flood the
// log line.
func (b *bootstrapper) summarizeTables(tables []domain.TableIdentifier, limit int) string {
	shown := min(len(tables), limit)

	names := make([]string, 0, shown)
	for _, t := range tables[:shown] {
		names = append(names, string(t.Schema)+"."+string(t.Name))
	}

	summary := strings.Join(names, ", ")
	if rest := len(tables) - shown; rest > 0 {
		summary += fmt.Sprintf(", ... and %d more", rest)
	}

	return summary
}

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

//go:embed sql/bootstrapper_replica_identity_ensure_full.sql
var bootstrapperReplicaIdentityEnsureFullSql string

//go:embed sql/bootstrapper_metadata_tables_timescale.sql
var bootstrapperMetadataTablesTimescaleSql string

//go:embed sql/bootstrapper_metadata_tables_no_timescale.sql
var bootstrapperMetadataTablesNoTimescaleSql string

//go:embed sql/bootstrapper_metadata_has_timescale.sql
var bootstrapperMetadataHasTimescaleSql string

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

// EnsureReplicaIdentityFull sets REPLICA IDENTITY FULL on every configured
// table and its partitions, skipping tables already at FULL. TimescaleDB chunks
// are left alone: they inherit from the hypertable and cannot be altered directly.
func (b *bootstrapper) EnsureReplicaIdentityFull(ctx context.Context, tables []domain.TableIdentifier) error {
	info, err := b.getTablesWithPartitions(ctx, tables)
	if err != nil {
		return err
	}

	seen := make(map[domain.TableIdentifier]struct{})
	var targets []domain.TableIdentifier
	add := func(table domain.TableIdentifier) {
		if _, ok := seen[table]; ok {
			return
		}
		seen[table] = struct{}{}
		targets = append(targets, table)
	}

	for table, t := range info {
		add(table)
		if t.Type == tableTypeHypertable {
			continue
		}
		for _, p := range t.Partitions {
			if p.Schema == "" || p.Name == "" {
				continue
			}
			add(p)
		}
	}

	// The embedded SQL is a no-op when the table is already FULL.
	for _, t := range targets {
		schemaLiteral := b.quoteLiteral(string(t.Schema))
		nameLiteral := b.quoteLiteral(string(t.Name))
		qualifiedIdentifier := b.quoteIdentifier(string(t.Schema)) + "." + b.quoteIdentifier(string(t.Name))

		sql := fmt.Sprintf(bootstrapperReplicaIdentityEnsureFullSql, schemaLiteral, nameLiteral, qualifiedIdentifier)

		if _, err := b.connection.Exec(ctx, sql); err != nil {
			return fmt.Errorf("set replica identity full on %s.%s: %w", t.Schema, t.Name, err)
		}
	}

	return nil
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

	schemas := make([]string, len(tables))
	names := make([]string, len(tables))
	for i, t := range tables {
		schemas[i] = string(t.Schema)
		names[i] = string(t.Name)
	}

	sql := bootstrapperMetadataTablesNoTimescaleSql
	if timescale {
		sql = bootstrapperMetadataTablesTimescaleSql
	}

	arrays := []struct {
		name   string
		values []string
	}{
		{name: "schemas", values: schemas},
		{name: "names", values: names},
	}

	m := b.newTypeMap()
	params := make([]param, 0, len(arrays))
	for _, array := range arrays {
		encoded, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](array.values), nil)
		if err != nil {
			return nil, fmt.Errorf("encode %s array: %w", array.name, err)
		}

		params = append(params, param{Value: encoded, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode})
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

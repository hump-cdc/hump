package catalog

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
	"hump/internal/source/postgres/connection"
	"hump/internal/source/postgres/schema"
	"hump/internal/types/introspector"
)

//go:embed sql/tables_timescale.sql
var tablesTimescaleSql string

//go:embed sql/tables_no_timescale.sql
var tablesNoTimescaleSql string

// GetTablesWithPartitions returns the configured tables along with their
// partition layout (for partitioned/hypertables). Returns ErrNoTablesConfigured
// if tables is empty, ErrTablesNotFound if any configured table is missing
// from the database (the error message lists every missing table).
func GetTablesWithPartitions(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) (schema.TablesWithPartitions, error) {
	if len(tables) == 0 {
		return nil, schema.ErrNoTablesConfigured
	}

	cfg, err := GetConfig(ctx, conn)
	if err != nil {
		return nil, err
	}

	schemas := make([]string, len(tables))
	names := make([]string, len(tables))
	for i, t := range tables {
		schemas[i] = string(t.Schema)
		names[i] = string(t.Name)
	}

	sql := tablesNoTimescaleSql
	if cfg.HasTimescaleDB {
		sql = tablesTimescaleSql
	}

	m := pgtype.NewMap()
	schemasArr, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](schemas), nil)
	if err != nil {
		return nil, fmt.Errorf("encode schemas array: %w", err)
	}
	namesArr, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](names), nil)
	if err != nil {
		return nil, fmt.Errorf("encode names array: %w", err)
	}

	queryResult, err := conn.ExecParams(
		ctx,
		sql,
		[]connection.Param{
			{Value: schemasArr, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode},
			{Value: namesArr, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode},
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("get tables with partitions: %w", err)
	}

	result := make(schema.TablesWithPartitions)
	for _, values := range queryResult.Rows {
		if len(values) < 5 {
			return nil, fmt.Errorf("get tables with partitions: expected 5 columns, got %d", len(values))
		}

		schemaName := domain.TableSchema(values[0])
		name := domain.TableName(values[1])
		tableType := schema.TableType(string(values[2]))
		partitionSchema := values[3]
		partitionTable := values[4]

		schemaMap, ok := result[schemaName]
		if !ok {
			schemaMap = make(map[domain.TableName]*schema.TableInfo)
			result[schemaName] = schemaMap
		}

		info, ok := schemaMap[name]
		if !ok {
			info = &schema.TableInfo{Type: tableType}
			schemaMap[name] = info
		}

		if partitionSchema != nil {
			p := schema.Partition{Schema: domain.TableSchema(partitionSchema)}
			if partitionTable != nil {
				p.Table = domain.TableName(partitionTable)
			}
			info.Partitions = append(info.Partitions, p)
		}
	}

	var missing []string
	for _, t := range tables {
		if _, ok := result[t.Schema][t.Name]; !ok {
			missing = append(missing, string(t.Schema)+"."+string(t.Name))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", schema.ErrTablesNotFound, strings.Join(missing, ", "))
	}

	return result, nil
}

// GetTableByPartition returns the parent table reference whose partition list
// contains (partitionSchema, partitionTable). Returns nil if no parent matches.
func GetTableByPartition(ctx context.Context, conn *connection.Connection, partitionSchema domain.TableSchema, partitionTable domain.TableName, tables []introspector.TableRef) (*introspector.TableRef, error) {
	source, err := GetTablesWithPartitions(ctx, conn, tables)
	if err != nil {
		return nil, err
	}

	for parentSchema, nameMap := range source {
		for name, info := range nameMap {
			for _, p := range info.Partitions {
				if p.Schema != partitionSchema {
					continue
				}
				if info.Type == schema.TableTypeHypertable || p.Table == partitionTable {
					return &introspector.TableRef{Schema: parentSchema, Name: name}, nil
				}
			}
		}
	}

	return nil, nil
}

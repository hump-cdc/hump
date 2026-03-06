// Package catalog reads PostgreSQL system catalog metadata: server settings,
// table layout (including partitioned tables and TimescaleDB hypertables),
// replication slots, publications, and replica identities.
package catalog

import (
	"context"

	"hump/internal/source/postgres/connection"
	"hump/internal/source/postgres/schema"
	"hump/internal/types/introspector"
)

// IntrospectedTable holds the resolved table kind together with the partitions
// that belong to it. Partitions is empty for non-partitioned tables.
type IntrospectedTable struct {
	Type       schema.TableType
	Partitions []schema.Partition
}

// TablesIntrospection is the reshaped result of IntrospectTables: a parent
// table view (Tables) and a flat partition→parent reverse index (Partitions).
type TablesIntrospection struct {
	// Tables maps schema -> table name -> table info with partitions.
	Tables map[introspector.TableSchema]map[introspector.TableName]*IntrospectedTable
	// Partitions maps partition schema -> partition name -> parent table reference.
	Partitions map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef
}

// IntrospectTables runs GetTablesWithPartitions and reshapes the result into a
// parent map plus a flat partition→parent reverse index.
func IntrospectTables(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) (*TablesIntrospection, error) {
	// Introspect database to get description of configured tables and their partitions

	dbTables, err := GetTablesWithPartitions(ctx, conn, tables)
	if err != nil {
		return nil, err
	}

	// Extract results

	result := &TablesIntrospection{
		Tables:     make(map[introspector.TableSchema]map[introspector.TableName]*IntrospectedTable),
		Partitions: make(map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef),
	}

	for schemaName, nameMap := range dbTables {
		tables, ok := result.Tables[schemaName]
		if !ok {
			tables = make(map[introspector.TableName]*IntrospectedTable)
			result.Tables[schemaName] = tables
		}

		for name, info := range nameMap {
			entry, ok := tables[name]
			if !ok {
				entry = &IntrospectedTable{Type: info.Type}
				tables[name] = entry
			}

			if info.Type != schema.TableTypePartitioned && info.Type != schema.TableTypeHypertable {
				continue
			}

			for _, p := range info.Partitions {
				if p.Table == "" {
					continue
				}

				entry.Partitions = append(entry.Partitions, schema.Partition{Schema: p.Schema, Table: p.Table})

				partSchema, ok := result.Partitions[p.Schema]
				if !ok {
					partSchema = make(map[introspector.TableName]introspector.TableRef)
					result.Partitions[p.Schema] = partSchema
				}
				partSchema[p.Table] = introspector.TableRef{Schema: schemaName, Name: name}
			}
		}
	}

	return result, nil
}

package schema

import "hump/internal/domain"

// TableType classifies how a PostgreSQL table participates in partitioning.
// Replication behavior (publication clauses, replica identity) varies by type.
type TableType string

const (
	// TableTypeRegular is a non-partitioned table.
	TableTypeRegular TableType = "regular"
	// TableTypeHypertable is a TimescaleDB hypertable whose chunks live in
	// TimescaleDB-managed schemas.
	TableTypeHypertable TableType = "hypertable"
	// TableTypePartitioned is a native PostgreSQL declaratively-partitioned table.
	TableTypePartitioned TableType = "partitioned"
)

// Partition identifies a single partition (or hypertable chunk) by its
// physical schema and table name.
type Partition struct {
	Schema domain.TableSchema
	Table  domain.TableName
}

// TableInfo describes a configured table: its classification and the set of
// partition tables that physically store its rows.
type TableInfo struct {
	Type       TableType
	Partitions []Partition
}

// TablesWithPartitions maps schema -> table name -> table info.
type TablesWithPartitions map[domain.TableSchema]map[domain.TableName]*TableInfo

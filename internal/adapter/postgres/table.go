package postgres

import "hump/internal/domain"

// tableType classifies how a PostgreSQL table participates in partitioning.
// Replication behavior (publication clauses, replica identity) varies by type.
type tableType int

const (
	// tableTypeRegular is a non-partitioned table, and the zero value.
	tableTypeRegular tableType = iota
	// tableTypeHypertable is a TimescaleDB hypertable whose chunks live in
	// TimescaleDB-managed schemas.
	tableTypeHypertable
	// tableTypePartitioned is a native PostgreSQL declaratively-partitioned table.
	tableTypePartitioned
)

// parseTableType maps a type literal from the table layout query to its
// tableType, falling back to tableTypeRegular for anything unrecognized.
func parseTableType(s string) tableType {
	switch s {
	case "hypertable":
		return tableTypeHypertable
	case "partitioned":
		return tableTypePartitioned
	default:
		return tableTypeRegular
	}
}

// tableMetadata describes a configured table: its classification and the set of
// partition tables that physically store its rows.
type tableMetadata struct {
	Type       tableType
	Partitions []domain.TableIdentifier
}

// tablesMetadata maps a table to its classification and partitions.
type tablesMetadata map[domain.TableIdentifier]*tableMetadata

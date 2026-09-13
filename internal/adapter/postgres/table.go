package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
)

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

// tableArrayParams encodes tables as the (schemas, names) text-array bind
// parameters every catalog query in this package filters on.
func tableArrayParams(m *pgtype.Map, tables []domain.TableIdentifier) ([]param, error) {
	schemas := make([]string, len(tables))
	names := make([]string, len(tables))
	for i, t := range tables {
		schemas[i] = string(t.Schema)
		names[i] = string(t.Name)
	}

	arrays := []struct {
		name   string
		values []string
	}{
		{name: "schemas", values: schemas},
		{name: "names", values: names},
	}

	params := make([]param, 0, len(arrays))
	for _, array := range arrays {
		encoded, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](array.values), nil)
		if err != nil {
			return nil, fmt.Errorf("encode %s array: %w", array.name, err)
		}

		params = append(params, param{Value: encoded, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode})
	}

	return params, nil
}

// tableScalarParams encodes table as the (schema, name) text bind parameters a
// catalog query filtering on a single table takes.
func tableScalarParams(table domain.TableIdentifier) []param {
	return []param{
		textParam(string(table.Schema)),
		textParam(string(table.Name)),
	}
}

// textParam encodes value as a text-format bind parameter.
func textParam(value string) param {
	return param{Value: []byte(value), OID: pgtype.TextOID, Format: pgtype.TextFormatCode}
}

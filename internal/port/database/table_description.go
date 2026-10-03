package database

import "hump/internal/domain"

// TableType classifies how a table stores its rows.
type TableType string

const (
	// TableTypeRegular is a plain, non-partitioned table.
	TableTypeRegular TableType = "regular"
	// TableTypePartitioned is a declaratively partitioned table.
	TableTypePartitioned TableType = "partitioned"
	// TableTypeHypertable is a TimescaleDB hypertable.
	TableTypeHypertable TableType = "hypertable"
)

// TableColumn is one column of a table.
type TableColumn struct {
	Name       string
	Type       string // Type is the column type as the server formats it, e.g. "timestamp with time zone".
	Nullable   bool
	PrimaryKey bool
}

// TableDescription is the layout of one table and the row filter suggested for its snapshot.
type TableDescription struct {
	Table   domain.TableIdentifier
	Type    TableType
	Columns []TableColumn
	Where   string // Where is a suggested filter that keeps recent rows; empty suggests none.
}

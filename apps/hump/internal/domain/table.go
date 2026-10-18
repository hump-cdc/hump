package domain

// TableSchema is the PostgreSQL schema a table lives in.
type TableSchema string

// TableName is a table name, unqualified by its schema.
type TableName string

// TableIdentifier is a schema-qualified table. It is used as a map key, so it
// must stay comparable.
type TableIdentifier struct {
	Schema TableSchema
	Name   TableName
}

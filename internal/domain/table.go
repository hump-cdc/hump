package domain

type TableSchema string

type TableName string

type TableIdentifier struct {
	Schema TableSchema
	Name   TableName
}

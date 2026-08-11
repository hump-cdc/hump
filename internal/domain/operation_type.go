package domain

type OperationType string

const (
	OperationTypeInsert   OperationType = "insert"
	OperationTypeUpdate   OperationType = "update"
	OperationTypeDelete   OperationType = "delete"
	OperationTypeTruncate OperationType = "truncate"
)

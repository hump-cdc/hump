package domain

type OperationType string

const (
	OperationInsert   OperationType = "insert"
	OperationUpdate   OperationType = "update"
	OperationDelete   OperationType = "delete"
	OperationTruncate OperationType = "truncate"
)

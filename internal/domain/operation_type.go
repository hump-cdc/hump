package domain

type OperationType string

const (
	OperationTypeInsert   OperationType = "insert"
	OperationTypeUpdate   OperationType = "update"
	OperationTypeDelete   OperationType = "delete"
	OperationTypeTruncate OperationType = "truncate"
)

// OperationTypes lists every operation type in canonical order.
var OperationTypes = []OperationType{
	OperationTypeInsert,
	OperationTypeUpdate,
	OperationTypeDelete,
	OperationTypeTruncate,
}

package domain

// OperationType is the kind of row change a WAL event carries. Its values are
// wire formats: sink YAML, the operation metric label and templates use them.
type OperationType string

// The operation types hump captures.
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

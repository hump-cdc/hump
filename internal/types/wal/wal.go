package wal

import (
	"context"
	"time"
)

type WalOperation string

var (
	WalOperationInsert   WalOperation = "insert"
	WalOperationUpdate   WalOperation = "update"
	WalOperationDelete   WalOperation = "delete"
	WalOperationTruncate WalOperation = "truncate"
)

type ProcessHandler func(
	ctx context.Context,
	timestamp time.Time,
	operation WalOperation,
	dbName string,
	schema string,
	table string,
	headers map[string]string,
	prev map[string]any,
	next map[string]any,
) (string, string, string, error)

type Wal interface {
	Start(ctx context.Context) error
	Close(ctx context.Context) error
}

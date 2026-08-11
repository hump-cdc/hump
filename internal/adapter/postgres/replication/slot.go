package replication

import (
	"context"
	_ "embed"
	"fmt"

	"hump/internal/adapter/postgres/connection"
)

//go:embed sql/slot.sql
var slotSql string

//go:embed sql/slot_remove.sql
var slotRemoveSql string

// EnsureReplicationSlotExists ensures the logical replication slot named
// slotName exists and uses the pgoutput plugin.
//
//   - If the slot does not exist, it is created with plugin pgoutput.
//   - If it exists with a different plugin, it is dropped (only possible
//     while inactive) and recreated with pgoutput.
//   - If it already exists with pgoutput, no action is taken.
func EnsureReplicationSlotExists(ctx context.Context, conn *connection.Connection, slotName string) error {
	sql := fmt.Sprintf(slotSql, connection.QuoteLiteral(slotName))

	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("ensure replication slot %q: %w", slotName, err)
	}

	return nil
}

// EnsureReplicationSlotRemoved drops the logical replication slot named slotName
// if it exists and is inactive. It is a no-op when the slot is absent and
// returns an error when the slot is currently active (held by another backend),
// since pg_drop_replication_slot cannot drop an active slot.
func EnsureReplicationSlotRemoved(ctx context.Context, conn *connection.Connection, slotName string) error {
	sql := fmt.Sprintf(slotRemoveSql, connection.QuoteLiteral(slotName))

	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("remove replication slot %q: %w", slotName, err)
	}

	return nil
}

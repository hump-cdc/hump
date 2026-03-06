package replication

import (
	"context"
	_ "embed"
	"fmt"

	"hump/internal/source/postgres/connection"
)

//go:embed sql/slot.sql
var slotSql string

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

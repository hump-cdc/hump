package postgres

import (
	"context"
	"fmt"
)

// Cleanup removes the publication and replication slot, leaving replica identity
// untouched. The slot is only dropped when inactive; an active one errors so a
// live consumer keeps its retained WAL.
func (p *Postgres) Cleanup(ctx context.Context) error {
	conn, err := newConnection(p.databaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	bs := newBootstrapper(conn)

	if err := bs.EnsurePublicationRemoved(ctx, p.publicationName); err != nil {
		return fmt.Errorf("remove publication: %w", err)
	}

	if err := bs.EnsureReplicationSlotRemoved(ctx, p.slotName); err != nil {
		return fmt.Errorf("remove replication slot: %w", err)
	}

	return nil
}

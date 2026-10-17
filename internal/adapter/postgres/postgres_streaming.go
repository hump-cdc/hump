package postgres

import (
	"context"
	"fmt"
	"sync"

	"hump/internal/port/database"
)

// Listen bootstraps the database over a control connection, then streams WAL
// over a second, replication-enabled one, dispatching decoded rows to handler.
// It blocks until ctx is cancelled or the stream fails.
func (p *Postgres) Listen(ctx context.Context, handler database.ProcessMessageHandler) error {
	conn, err := newConnection(p.databaseUrl)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer closeDetached(conn.Close)

	introspector, err := p.configure(ctx, conn)
	if err != nil {
		return fmt.Errorf("unable to configure to database: %w", err)
	}

	p.healthPostgres.MarkHealthy()

	// The poller gets its own session: conn serves the introspector from the WAL
	// loop, and a pgconn session cannot run two statements at once.
	lagConnection, err := newConnection(p.databaseUrl)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer closeDetached(lagConnection.Close)

	lagCtx, stopLag := context.WithCancel(ctx)
	var lagDone sync.WaitGroup
	lagDone.Go(func() { newReplicationLagPoller(lagConnection, p.slotName, p.metrics).run(lagCtx) })
	defer lagDone.Wait()
	defer stopLag()

	replicationConnection, err := newConnection(p.databaseUrl, withReplication(true))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer closeDetached(replicationConnection.Close)

	w, err := newWal(ctx, replicationConnection, p.publicationName, p.slotName, introspector, p.gate, p.healthWal, p.metrics)
	if err != nil {
		return fmt.Errorf("failed to initialize WAL processing: %w", err)
	}

	err = w.Start(ctx, handler)
	defer closeDetached(w.Stop)

	return err
}

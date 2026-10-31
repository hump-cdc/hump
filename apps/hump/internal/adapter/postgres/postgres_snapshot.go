package postgres

import (
	"context"

	"hump/internal/domain"
	"hump/internal/port/database"
)

// ValidateWhere reports whether where is a row filter a snapshot accepts.
func (p *Postgres) ValidateWhere(where string) error {
	return p.snapshot.ValidateWhere(where)
}

// StartSnapshot pauses WAL delivery and exports a snapshot of tables, returning
// its id, or the active snapshot's id while one is open.
func (p *Postgres) StartSnapshot(ctx context.Context, tables []domain.TableIdentifier) (string, error) {
	return p.snapshot.Start(ctx, tables)
}

// FinishSnapshot releases the active snapshot and resumes WAL delivery.
func (p *Postgres) FinishSnapshot(ctx context.Context) error {
	return p.snapshot.Finish(ctx)
}

// ReadSnapshot calls onRow for every row of table matching where, as the
// snapshot with snapshotID sees it.
func (p *Postgres) ReadSnapshot(ctx context.Context, snapshotID string, table domain.TableIdentifier, where string, onRow func(domain.DbRecord, database.SnapshotRowMetadata) error) error {
	return p.snapshot.Read(ctx, snapshotID, table, where, onRow)
}

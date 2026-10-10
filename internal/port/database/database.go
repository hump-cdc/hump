package database

import (
	"context"
	"hump/internal/domain"
	"time"
)

type Database interface {
	// GetDbName returns the database name, or an empty string when it is unknown.
	GetDbName() string
	// GetDbHost returns the database host, or an empty string when it is unknown.
	GetDbHost() string
	// GetDbPort returns the database port, or zero when it is unknown.
	GetDbPort() uint16
	// GetPublicationName returns the publication changes are replicated through.
	GetPublicationName() string
	// GetSlotName returns the replication slot changes are streamed from.
	GetSlotName() string
	// Listen prepares the publication and replication slot, then streams changes
	// to handler. It blocks until ctx is cancelled or the stream fails.
	Listen(ctx context.Context, handler ProcessMessageHandler) error
	// Cleanup removes the publication and replication slot. It fails rather than
	// drop a slot that is still in use.
	Cleanup(ctx context.Context) error
	// GetTables describes tables. Tables missing from the database are skipped.
	GetTables(ctx context.Context, tables []domain.TableIdentifier) ([]TableDescription, error)
	// GetRowCount streams counts of the rows of table matching where condition.
	// An invalid where returns an error wrapping ErrInvalidWhere.
	GetRowCount(ctx context.Context, table domain.TableIdentifier, where string) (<-chan RowCount, error)
	// ValidateWhere validates the where condition, returning an error wrapping
	// ErrInvalidWhere when it is invalid.
	ValidateWhere(where string) error
	// StartSnapshot pauses WAL processing and exports a snapshot that keeps
	// tables locked against rewrites, returning its id. While a snapshot is
	// active it returns the same id again.
	StartSnapshot(ctx context.Context, tables []domain.TableIdentifier) (string, error)
	// FinishSnapshot releases the active snapshot and resumes WAL processing.
	// It is a no-op without an active snapshot.
	FinishSnapshot(ctx context.Context) error
	// ReadSnapshot reads every row of table matching where as the snapshot with
	// snapshotID sees it, calling onRow for each; an onRow error stops the read.
	ReadSnapshot(ctx context.Context, snapshotID string, table domain.TableIdentifier, where string, onRow func(domain.DbRecord) error) error
}

type ProcessMessageHandler func(
	ctx context.Context,

	timestamp time.Time,
	operation domain.OperationType,

	dbName domain.DbName,
	schema domain.TableSchema,
	table domain.TableName,

	headers domain.Headers,
	prev domain.DbRecord,
	next domain.DbRecord,
) error

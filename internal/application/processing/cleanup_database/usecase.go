// Package cleanupdatabase drops the publication and the replication slot hump
// created, so the database stops retaining WAL for it.
package cleanupdatabase

import (
	"context"
	"hump/internal/port/database"
)

// UseCase removes hump's replication objects from the source database.
type UseCase struct {
	db database.Database
}

// NewUseCase returns a use case cleaning up db.
func NewUseCase(db database.Database) *UseCase {
	return &UseCase{
		db: db,
	}
}

// Cleanup drops the publication and the replication slot of the database.
func (uc *UseCase) Cleanup(ctx context.Context) error {
	if err := uc.db.Cleanup(ctx); err != nil {
		return err
	}

	logger.Info("Cleanup complete", "publication", uc.db.GetPublicationName(), "slot", uc.db.GetSlotName())

	return nil
}

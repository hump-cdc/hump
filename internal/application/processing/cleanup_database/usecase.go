package cleanupdatabase

import (
	"context"
	"hump/internal/port/database"
)

type UseCase struct {
	db database.Database
}

func NewUseCase(db database.Database) *UseCase {
	return &UseCase{
		db: db,
	}
}

func (uc *UseCase) Cleanup(ctx context.Context) error {
	if err := uc.db.Cleanup(ctx); err != nil {
		return err
	}

	logger.Info("Cleanup complete", "publication", uc.db.GetPublicationName(), "slot", uc.db.GetSlotName())

	return nil
}

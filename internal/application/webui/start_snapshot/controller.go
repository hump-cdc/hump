// Package startsnapshot starts a snapshot run of the tables the web UI chose.
package startsnapshot

import (
	"context"
	"errors"

	"hump/internal/domain"
	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2"
)

// Controller serves POST /api/snapshot/start.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller starting snapshots through useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Handler returns the huma handler starting a run over the requested tables.
func (c *Controller) Handler() func(ctx context.Context, input *SnapshotStartRequest) (*struct{}, error) {
	return func(_ context.Context, input *SnapshotStartRequest) (*struct{}, error) {
		err := c.useCase.Start(requests(input.Body.Tables))
		switch {
		case errors.Is(err, snapshot.ErrAlreadyRunning):
			return nil, huma.Error409Conflict(err.Error())
		case errors.Is(err, snapshot.ErrInvalidWhere):
			return nil, huma.Error422UnprocessableEntity(err.Error())
		case errors.Is(err, ErrUnavailable):
			return nil, huma.Error503ServiceUnavailable(err.Error())
		case err != nil:
			return nil, err
		}

		return nil, nil
	}
}

// requests converts the requested tables to the use case's vocabulary.
func requests(tables []SnapshotStartTable) []snapshot.Request {
	result := make([]snapshot.Request, 0, len(tables))
	for _, table := range tables {
		result = append(result, snapshot.Request{
			Table: domain.TableIdentifier{Schema: domain.TableSchema(table.Schema), Name: domain.TableName(table.Table)},
			Where: table.Where,
		})
	}

	return result
}

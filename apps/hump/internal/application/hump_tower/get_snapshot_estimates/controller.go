// Package getsnapshotestimates serves the snapshot estimates stream of the
// web UI: row counts of the requested tables, streamed as they arrive.
package getsnapshotestimates

import (
	"context"

	"hump/internal/domain"

	"github.com/danielgtaylor/huma/v2/sse"
)

// Controller serves POST /api/snapshot/estimate.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller streaming the estimates of useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Events maps every event name the stream sends to its payload type.
func (c *Controller) Events() map[string]any {
	return map[string]any{
		"estimate": EstimateEvent{},
		"error":    EstimateErrorEvent{},
		"done":     EstimatesDoneEvent{},
	}
}

// Handler returns the SSE handler streaming row counts as they arrive.
func (c *Controller) Handler() func(ctx context.Context, input *EstimatesRequest, send sse.Sender) {
	return func(ctx context.Context, input *EstimatesRequest, send sse.Sender) {
		// Returning cancels ctx, which is what stops the use case's goroutines
		// when the client is gone: they never wait on a reader that left.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for estimate := range c.useCase.Estimate(ctx, requests(input.Body.Tables)) {
			if err := send.Data(event(estimate)); err != nil {
				return
			}
		}

		_ = send.Data(EstimatesDoneEvent{})
	}
}

// requests converts the requested tables to the use case's vocabulary.
func requests(tables []EstimateTable) []Request {
	result := make([]Request, 0, len(tables))
	for _, table := range tables {
		result = append(result, Request{
			Table: domain.TableIdentifier{Schema: domain.TableSchema(table.Schema), Name: domain.TableName(table.Table)},
			Where: table.Where,
		})
	}

	return result
}

// event renders one estimate as the event the client receives.
func event(estimate Estimate) any {
	schema, table := string(estimate.Table.Schema), string(estimate.Table.Name)

	if estimate.Error != nil {
		return EstimateErrorEvent{Schema: schema, Table: table, Error: estimate.Error.Error()}
	}

	return EstimateEvent{Schema: schema, Table: table, Count: estimate.Count, Rough: estimate.Rough}
}

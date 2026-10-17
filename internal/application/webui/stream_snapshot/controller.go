// Package streamsnapshot serves the snapshot stream of the web UI: the state
// of the snapshot run on connect, then again on every change.
package streamsnapshot

import (
	"context"

	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2/sse"
)

type Controller struct {
	useCase *UseCase
}

// NewController returns a controller streaming the snapshot state of useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Events maps every event name the stream sends to its payload type.
func (c *Controller) Events() map[string]any {
	return map[string]any{
		"idle":       SnapshotIdleEvent{},
		"processing": SnapshotProcessingEvent{},
		"done":       SnapshotDoneEvent{},
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}, send sse.Sender) {
	return func(ctx context.Context, _ *struct{}, send sse.Sender) {
		state, updates, release := c.useCase.Subscribe()
		defer release()

		if err := send.Data(event(state)); err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return

			case state, ok := <-updates:
				// A closed channel is the runner shutting down; a done ctx is
				// the client leaving. Both end the stream.
				if !ok {
					return
				}

				if err := send.Data(event(state)); err != nil {
					return
				}
			}
		}
	}
}

// event renders one state as the event the client receives.
func event(state snapshot.State) any {
	switch state.Status {
	case snapshot.StatusProcessing:
		tables := make([]SnapshotProcessingTable, 0, len(state.Tables))
		for _, table := range state.Tables {
			tables = append(tables, SnapshotProcessingTable{
				Schema:        string(table.Table.Schema),
				Name:          string(table.Table.Name),
				Where:         table.Where,
				Status:        string(table.Status),
				CompletedRows: table.CompletedRows,
				EstimatedRows: table.EstimatedRows,
				Sinks:         sinks(table.Sinks),
			})
		}

		return SnapshotProcessingEvent{State: string(state.Status), StartedAt: state.StartedAt, Tables: tables}

	case snapshot.StatusDone:
		tables := make([]SnapshotDoneTable, 0, len(state.Tables))
		for _, table := range state.Tables {
			tables = append(tables, SnapshotDoneTable{
				Schema:        string(table.Table.Schema),
				Name:          string(table.Table.Name),
				Status:        string(table.Status),
				Error:         table.Error,
				CompletedRows: table.CompletedRows,
				Sinks:         sinks(table.Sinks),
			})
		}

		return SnapshotDoneEvent{State: string(state.Status), StartedAt: state.StartedAt, FinishedAt: state.FinishedAt, Tables: tables}

	default:
		return SnapshotIdleEvent{State: string(snapshot.StatusIdle)}
	}
}

// sinks renders the sinks of one table.
func sinks(source []snapshot.Sink) []SnapshotRunSink {
	result := make([]SnapshotRunSink, 0, len(source))
	for _, s := range source {
		result = append(result, SnapshotRunSink{Type: s.Type, Host: s.Host, Target: s.Target, Count: s.Count})
	}

	return result
}

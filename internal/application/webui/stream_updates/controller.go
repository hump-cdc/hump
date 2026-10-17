// Package streamupdates serves the live updates stream of the web UI: one
// event per batching window, listing the deliveries the pipeline made in it,
// and the snapshot state whenever it changes. A connected client is what turns
// the collectors on.
package streamupdates

import (
	"context"

	livestream "hump/internal/port/live_stream"
	"hump/internal/port/snapshot"

	"github.com/danielgtaylor/huma/v2/sse"
)

type Controller struct {
	useCase *UseCase
}

// NewController returns a controller streaming the live feed of useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Events maps every event name the stream sends to its payload type.
func (c *Controller) Events() map[string]any {
	return map[string]any{
		"welcome":    WelcomeEvent{},
		"operations": OperationsEvent{},
		"snapshot":   SnapshotEvent{},
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}, send sse.Sender) {
	return func(ctx context.Context, _ *struct{}, send sse.Sender) {
		feed, release := c.useCase.Subscribe()
		defer release()

		if err := send.Data(WelcomeEvent{}); err != nil {
			return
		}

		last := snapshotEvent(feed.Snapshot)
		if err := send.Data(last); err != nil {
			return
		}

		snapshots := feed.Snapshots

		for {
			select {
			case <-ctx.Done():
				return

			case batch, ok := <-feed.Batches:
				// A closed channel is the streamer shutting down; a done ctx is
				// the client leaving. Both end the stream.
				if !ok {
					return
				}

				if err := send.Data(operationsEvent(batch)); err != nil {
					return
				}

			case state, ok := <-snapshots:
				// The runner shutting down does not end the stream: the feed
				// closes on its own, and a nil channel just stops this case.
				if !ok {
					snapshots = nil
					continue
				}

				// The runner publishes far more often than the percent moves;
				// the client only hears about a new state or a new percent.
				next := snapshotEvent(state)
				if next == last {
					continue
				}

				if err := send.Data(next); err != nil {
					return
				}

				last = next
			}
		}
	}
}

// operationsEvent renders one window as the event the client receives.
func operationsEvent(batch livestream.Batch) OperationsEvent {
	operations := make([]OperationEvent, 0, len(batch.Operations))
	for _, op := range batch.Operations {
		operations = append(operations, OperationEvent{
			Schema: op.Schema,
			Table:  op.Table,
			Sink:   op.Sink,
			Host:   op.Host,
			Target: op.Target,
		})
	}

	return OperationsEvent{Operations: operations}
}

// snapshotEvent renders one snapshot state as the event the client receives.
// An unknown status is reported as idle.
func snapshotEvent(state snapshot.State) SnapshotEvent {
	switch state.Status {
	case snapshot.StatusProcessing, snapshot.StatusDone:
		return SnapshotEvent{State: string(state.Status), Progress: progress(state)}

	default:
		return SnapshotEvent{State: string(snapshot.StatusIdle)}
	}
}

// Package streamupdates serves the live updates stream of the web UI: one
// event per batching window, listing the deliveries the pipeline made in it. A
// connected client is what turns the collectors on.
package streamupdates

import (
	"context"

	livestream "hump/internal/port/live_stream"

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
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}, send sse.Sender) {
	return func(ctx context.Context, _ *struct{}, send sse.Sender) {
		batches, release := c.useCase.Subscribe()
		defer release()

		if err := send.Data(WelcomeEvent{}); err != nil {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return

			case batch, ok := <-batches:
				// A closed channel is the streamer shutting down; a done ctx is
				// the client leaving. Both end the stream.
				if !ok {
					return
				}

				if err := send.Data(operationsEvent(batch)); err != nil {
					return
				}
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

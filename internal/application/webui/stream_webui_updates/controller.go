// Package streamwebuiupdates serves the live updates stream of the web UI: one
// event per batching window, listing the deliveries the pipeline made in it. A
// connected client is what turns the collectors on.
package streamwebuiupdates

import (
	"context"

	livestream "hump/internal/port/live_stream"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"

	"github.com/danielgtaylor/huma/v2/sse"
)

type Controller struct {
	metrics    metricscollector.Collector    // metrics is nil when telemetry is disabled.
	operations operationscollector.Collector // operations is nil when the web UI is disabled.
	stream     livestream.Streamer           // stream is nil when the web UI is disabled.
}

// NewController returns a controller that holds a collector connection and a
// live-feed subscription per client. Every dependency is nil when the feature
// behind it is disabled.
func NewController(
	metrics metricscollector.Collector,
	operations operationscollector.Collector,
	stream livestream.Streamer,
) *Controller {
	return &Controller{
		metrics:    metrics,
		operations: operations,
		stream:     stream,
	}
}

// Events maps every event name the stream sends to its payload type.
func (c *Controller) Events() map[string]any {
	return map[string]any{
		"operations": OperationsEvent{},
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}, send sse.Sender) {
	return func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if c.metrics != nil {
			c.metrics.Connect()
			defer c.metrics.Disconnect()
		}

		// The operations log is idle until a client connects, so nothing is
		// recorded — and nothing reaches the feed below — without this.
		if c.operations != nil {
			c.operations.Connect()
			defer c.operations.Disconnect()
		}

		// A nil channel blocks forever, which is what a disabled feed should do.
		var batches <-chan livestream.Batch
		if c.stream != nil {
			id, ch := c.stream.Subscribe()
			defer c.stream.Unsubscribe(id)
			batches = ch
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

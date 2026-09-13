// Package streamwebuiupdates serves the live updates stream of the web UI. For
// now it only keeps the connection alive and marks the client as connected to
// the metrics collector.
package streamwebuiupdates

import (
	"context"
	"time"

	metricscollector "hump/internal/port/metrics_collector"

	"github.com/danielgtaylor/huma/v2/sse"
)

// pingInterval is how often a ping is sent to keep the connection alive.
const pingInterval = 15 * time.Second

// PingEvent is a keepalive event.
type PingEvent struct {
	Timestamp time.Time `json:"timestamp" doc:"Time the ping was sent"`
}

type Controller struct {
	collector metricscollector.Collector
}

// NewController returns a controller that holds a collector connection per
// client; collector is nil when telemetry is disabled.
func NewController(collector metricscollector.Collector) *Controller {
	return &Controller{
		collector: collector,
	}
}

// Events maps every event name the stream sends to its payload type.
func (c *Controller) Events() map[string]any {
	return map[string]any{
		"ping": PingEvent{},
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}, send sse.Sender) {
	return func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if c.collector != nil {
			c.collector.Connect()
			defer c.collector.Disconnect()
		}

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		// The first ping flushes the response headers, so the client sees the
		// stream open right away.
		for {
			if err := send.Data(PingEvent{Timestamp: time.Now()}); err != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}
}

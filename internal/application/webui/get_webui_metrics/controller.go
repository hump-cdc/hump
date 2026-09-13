package getwebuimetrics

import (
	"context"
	"math"

	metricscollector "hump/internal/port/metrics_collector"

	"github.com/danielgtaylor/huma/v2"
)

type Controller struct {
	collector metricscollector.Collector
}

// NewController returns a controller serving the points of collector, which is
// nil when telemetry is disabled.
func NewController(collector metricscollector.Collector) *Controller {
	return &Controller{
		collector: collector,
	}
}

func (c *Controller) Handler() func(ctx context.Context, input *MetricsRequest) (*MetricsResponse, error) {
	return func(_ context.Context, input *MetricsRequest) (*MetricsResponse, error) {
		if c.collector == nil {
			return nil, huma.Error503ServiceUnavailable("telemetry disabled")
		}

		duration := metricscollector.Duration(input.Body.Duration)
		width, err := duration.Duration()
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		points, err := c.collector.Dump(duration)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		resp := MetricsResponse{}
		resp.Body.Duration = string(duration)
		resp.Body.BucketSeconds = int(width.Seconds())
		resp.Body.Points = make([]MetricsPoint, 0, len(points))

		for _, p := range points {
			sinkDelivery := make(map[string]int64, len(p.SinkDelivery))
			for sinkType, count := range p.SinkDelivery {
				sinkDelivery[sinkType] = toCount(count)
			}

			resp.Body.Points = append(resp.Body.Points, MetricsPoint{
				Timestamp:             p.Timestamp,
				Delivered:             toCount(p.Delivered),
				Failed:                toCount(p.Failed),
				Throughput:            p.Throughput,
				ReplicationLagBytes:   p.ReplicationLagBytes,
				MemoryAllocBytes:      p.MemoryAllocBytes,
				SinkDelivery:          sinkDelivery,
				SinkLatencyP95Seconds: p.SinkLatencyP95Seconds,
			})
		}

		return &resp, nil
	}
}

// toCount rounds a counter delta, which is whole but carried as a float, to an integer.
func toCount(value float64) int64 {
	return int64(math.Round(value))
}

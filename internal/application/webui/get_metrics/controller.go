// Package getmetrics serves the metrics of the web UI: bucketed series,
// lifetime counts per table and per sink, and the table-to-target edges.
package getmetrics

import (
	"context"
	"errors"
	"math"

	metricscollector "hump/internal/port/metrics_collector"

	"github.com/danielgtaylor/huma/v2"
)

// Controller serves POST /api/metrics.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller serving the metrics of useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Handler returns the huma handler rendering the metrics for the requested width.
func (c *Controller) Handler() func(ctx context.Context, input *MetricsRequest) (*MetricsResponse, error) {
	return func(_ context.Context, input *MetricsRequest) (*MetricsResponse, error) {
		metrics, err := c.useCase.Metrics(input.Body.Duration)
		if errors.Is(err, ErrDisabled) {
			return nil, huma.Error503ServiceUnavailable(err.Error())
		}
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		resp := MetricsResponse{}
		resp.Body.Duration = string(metrics.Duration)
		resp.Body.BucketSeconds = int(metrics.Bucket.Seconds())
		resp.Body.Points = c.points(metrics.Snapshot.Points)
		resp.Body.Tables = c.tables(metrics.Snapshot.Tables)
		resp.Body.Sinks = c.sinks(metrics.Snapshot.Sinks)
		resp.Body.Connections = c.connections(metrics.Snapshot.Connections)

		return &resp, nil
	}
}

// points renders the bucketed series.
func (c *Controller) points(source []metricscollector.Point) []MetricsPoint {
	result := make([]MetricsPoint, 0, len(source))

	for _, p := range source {
		sinkDelivery := make(map[string]int64, len(p.SinkDelivery))
		for sinkType, count := range p.SinkDelivery {
			sinkDelivery[sinkType] = toCount(count)
		}

		result = append(result, MetricsPoint{
			Timestamp:           p.Timestamp,
			Delivered:           toCount(p.Delivered),
			Failed:              toCount(p.Failed),
			Throughput:          p.Throughput,
			ReplicationLagBytes: p.ReplicationLagBytes,
			MemoryAllocBytes:    p.MemoryAllocBytes,
			SinkDelivery:        sinkDelivery,
			SinkLatencyP95:      p.SinkLatencyP95,
		})
	}

	return result
}

// tables renders the row operations by schema-qualified table name.
func (c *Controller) tables(source map[string]metricscollector.TableOps) map[string]MetricsTableOps {
	result := make(map[string]MetricsTableOps, len(source))

	for name, ops := range source {
		result[name] = MetricsTableOps{
			Insert:   tableOp(ops.Insert),
			Update:   tableOp(ops.Update),
			Delete:   tableOp(ops.Delete),
			Truncate: tableOp(ops.Truncate),
		}
	}

	return result
}

// tableOp renders the count and latency of one operation kind of a table.
func tableOp(op metricscollector.TableOp) MetricsTableOp {
	return MetricsTableOp{Count: toCount(op.Count), P95: op.P95}
}

// sinks renders the sink endpoints along with their targets.
func (c *Controller) sinks(source []metricscollector.Sink) []MetricsSink {
	result := make([]MetricsSink, 0, len(source))

	for _, s := range source {
		targets := make([]MetricsSinkTarget, 0, len(s.Targets))
		for _, t := range s.Targets {
			targets = append(targets, MetricsSinkTarget{
				Target:     t.Target,
				Operations: toCount(t.Operations),
				Success:    t.Success,
				P95:        t.P95,
			})
		}

		result = append(result, MetricsSink{
			Type:       s.Type,
			Host:       s.Host,
			Operations: toCount(s.Operations),
			Success:    s.Success,
			P95:        s.P95,
			Targets:    targets,
		})
	}

	return result
}

// connections renders the table-to-target edges of the pipeline.
func (c *Controller) connections(source []metricscollector.ConnectionOps) []MetricsConnection {
	result := make([]MetricsConnection, 0, len(source))

	for _, conn := range source {
		result = append(result, MetricsConnection{
			Table:   string(conn.Table.Schema) + "." + string(conn.Table.Name),
			Sink:    conn.SinkType,
			Host:    conn.Host,
			Target:  conn.Target,
			Success: toCount(conn.Success),
			Failure: toCount(conn.Failure),
		})
	}

	return result
}

// toCount rounds a counter delta, which is whole but carried as a float, to an integer.
func toCount(value float64) int64 {
	return int64(math.Round(value))
}

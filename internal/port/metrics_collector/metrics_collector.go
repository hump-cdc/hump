// Package metricscollector defines the on-demand collector that aggregates the
// process metrics shown on the web UI dashboard into fixed-width time buckets.
package metricscollector

import (
	"time"

	"hump/internal/domain"
)

// Duration is the width of the buckets a series of points is aggregated into.
type Duration string

const (
	// Duration1m aggregates points into one-minute buckets.
	Duration1m Duration = "1m"
	// Duration5m aggregates points into five-minute buckets.
	Duration5m Duration = "5m"
	// Duration10m aggregates points into ten-minute buckets.
	Duration10m Duration = "10m"
)

// Durations lists every supported bucket width.
var Durations = []Duration{Duration1m, Duration5m, Duration10m}

// Duration returns the bucket width d stands for, or [ErrUnknownDuration].
func (d Duration) Duration() (time.Duration, error) {
	switch d {
	case Duration1m:
		return time.Minute, nil
	case Duration5m:
		return 5 * time.Minute, nil
	case Duration10m:
		return 10 * time.Minute, nil
	default:
		return 0, ErrUnknownDuration
	}
}

// Point is the aggregate of every scrape that fell into one bucket.
type Point struct {
	// Timestamp is the start of the bucket.
	Timestamp time.Time
	// Delivered and Failed count the messages sinks finished during the bucket.
	Delivered float64
	Failed    float64
	// Throughput is Delivered per second of the time the bucket was scraped for.
	Throughput float64
	// ReplicationLagBytes and MemoryAllocBytes are the peak gauge readings.
	ReplicationLagBytes float64
	MemoryAllocBytes    float64
	// SinkDelivery counts delivered and failed messages by sink type.
	SinkDelivery map[string]float64
	// SinkLatencyP95 is the p95 write latency in milliseconds by sink type; a
	// type with no estimate for the bucket is absent.
	SinkLatencyP95 map[string]float64
}

// Connection identifies one table-to-sink-target edge of the pipeline.
type Connection struct {
	Table    domain.TableIdentifier
	SinkType string
	Host     string
	Target   string
}

// TableOps reports the row operations replicated from one table, by operation.
type TableOps struct {
	Insert   TableOp
	Update   TableOp
	Delete   TableOp
	Truncate TableOp
}

// TableOp is what one operation kind of one table replicated.
type TableOp struct {
	// Count is the number of rows replicated since the process started.
	Count float64
	// P95 is the p95 latency of processing a row through every sink, in
	// milliseconds, over the requested window; it is nil when the window
	// observed nothing.
	P95 *float64
}

// Delivery is what one sink, or one of its targets, delivered since the process
// started. [domain.DeliveryStatusRetry] is an attempt rather than an outcome and
// is left out.
type Delivery struct {
	Operations float64
	// Success is the share of operations that succeeded, 0 to 1; it is nil when
	// there was no operation to divide by.
	Success *float64
	// P95 is the p95 latency of a single write attempt, in milliseconds, over
	// the requested window; it is nil when the window observed nothing.
	P95 *float64
}

// Target is one rendered destination of a sink — a topic, subject, exchange,
// stream or HTTP method.
type Target struct {
	Target string
	Delivery
}

// Sink is one sink endpoint, identified by its type and connection host.
type Sink struct {
	Type    string
	Host    string
	Targets []Target
	Delivery
}

// ConnectionOps counts the deliveries that flowed along one connection.
type ConnectionOps struct {
	Connection
	Success float64
	Failure float64
}

// Snapshot is everything one Dump reports: the bucketed points, plus the flat
// views of the tables, sinks and connections of the pipeline.
type Snapshot struct {
	// Points holds the newest points of the requested duration, oldest first.
	Points []Point
	// Tables counts the row operations of every configured table, keyed by
	// "schema.table".
	Tables map[string]TableOps
	// Sinks and Connections cover the configured topology, so an entry with no
	// traffic is still present, with zero counts.
	Sinks       []Sink
	Connections []ConnectionOps
}

// Collector scrapes metrics while at least one client is connected, and for an
// idle period after the last one leaves.
type Collector interface {
	// Connect registers a client, activating the collector if it was idle.
	Connect()
	// Disconnect unregisters a client previously registered with Connect.
	Disconnect()
	// Scrape requests an extra scrape out of schedule; it does nothing while idle.
	Scrape()
	// Record counts n deliveries over conn that ended with status. Unlike the
	// scraped counts, it is reported by the pipeline rather than read from the
	// registry, and it is safe to call while the collector is idle.
	Record(conn Connection, status domain.DeliveryStatus, n int)
	// Dump returns the newest snapshot aggregated by d.
	Dump(d Duration) (Snapshot, error)
}

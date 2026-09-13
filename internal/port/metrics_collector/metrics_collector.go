// Package metricscollector defines the on-demand collector that aggregates the
// process metrics shown on the web UI dashboard into fixed-width time buckets.
package metricscollector

import "time"

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
	// SinkLatencyP95Seconds is the p95 write latency by sink type; a type with no
	// estimate for the bucket is absent.
	SinkLatencyP95Seconds map[string]float64
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
	// Dump returns the newest points aggregated by d, oldest first.
	Dump(d Duration) ([]Point, error)
}

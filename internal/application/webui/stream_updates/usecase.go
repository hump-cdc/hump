package streamupdates

import (
	livestream "hump/internal/port/live_stream"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"
	"hump/internal/port/snapshot"
)

// Feed is what one client is attached to.
type Feed struct {
	Batches   <-chan livestream.Batch // Batches is nil when the live feed is disabled.
	Snapshot  snapshot.State          // Snapshot is the state at subscribe time; idle without a runner.
	Snapshots <-chan snapshot.State   // Snapshots is nil without a runner.
}

type UseCase struct {
	metrics    metricscollector.Collector    // metrics is nil when telemetry is disabled.
	operations operationscollector.Collector // operations is nil when the web UI is disabled.
	stream     livestream.Streamer           // stream is nil when the web UI is disabled.
	runner     snapshot.Runner               // runner is nil in cleanup mode.
}

// NewUseCase returns a use case that holds a collector connection, a live-feed
// subscription and a snapshot subscription per client. Every dependency is nil
// when the feature behind it is disabled.
func NewUseCase(
	metrics metricscollector.Collector,
	operations operationscollector.Collector,
	stream livestream.Streamer,
	runner snapshot.Runner,
) *UseCase {
	return &UseCase{
		metrics:    metrics,
		operations: operations,
		stream:     stream,
		runner:     runner,
	}
}

// Subscribe attaches one client: it turns the collectors on and subscribes to
// the live feed and the snapshot runner, returning the feed and the func that
// undoes all of it. A disabled source is a nil channel, which blocks forever.
func (uc *UseCase) Subscribe() (Feed, func()) {
	if uc.metrics != nil {
		uc.metrics.Connect()
	}

	// The operations log is idle until a client connects, so nothing is
	// recorded — and nothing reaches the feed — without this.
	if uc.operations != nil {
		uc.operations.Connect()
	}

	feed := Feed{Snapshot: snapshot.State{Status: snapshot.StatusIdle}}

	var streamID uint64
	if uc.stream != nil {
		streamID, feed.Batches = uc.stream.Subscribe()
	}

	var runnerID uint64
	if uc.runner != nil {
		runnerID, feed.Snapshot, feed.Snapshots = uc.runner.Subscribe()
	}

	return feed, func() {
		if uc.runner != nil {
			uc.runner.Unsubscribe(runnerID)
		}

		if uc.stream != nil {
			uc.stream.Unsubscribe(streamID)
		}

		if uc.operations != nil {
			uc.operations.Disconnect()
		}

		if uc.metrics != nil {
			uc.metrics.Disconnect()
		}
	}
}

// progress returns the rows read across all tables as a whole percent of the
// estimate, rounded down. It is 0 outside processing and while the estimate is
// unknown.
func progress(state snapshot.State) uint8 {
	if state.Status != snapshot.StatusProcessing {
		return 0
	}

	var completed, estimated uint64
	for _, table := range state.Tables {
		completed += table.CompletedRows
		estimated += table.EstimatedRows
	}

	if estimated == 0 {
		return 0
	}

	return uint8(min(100, completed*100/estimated))
}

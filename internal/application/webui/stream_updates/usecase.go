package streamupdates

import (
	livestream "hump/internal/port/live_stream"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"
)

type UseCase struct {
	metrics    metricscollector.Collector    // metrics is nil when telemetry is disabled.
	operations operationscollector.Collector // operations is nil when the web UI is disabled.
	stream     livestream.Streamer           // stream is nil when the web UI is disabled.
}

// NewUseCase returns a use case that holds a collector connection and a
// live-feed subscription per client. Every dependency is nil when the feature
// behind it is disabled.
func NewUseCase(
	metrics metricscollector.Collector,
	operations operationscollector.Collector,
	stream livestream.Streamer,
) *UseCase {
	return &UseCase{
		metrics:    metrics,
		operations: operations,
		stream:     stream,
	}
}

// Subscribe attaches one client: it turns the collectors on and subscribes to
// the live feed, returning the feed and the func that undoes both. A disabled
// feed is a nil channel, which blocks forever.
func (uc *UseCase) Subscribe() (<-chan livestream.Batch, func()) {
	if uc.metrics != nil {
		uc.metrics.Connect()
	}

	// The operations log is idle until a client connects, so nothing is
	// recorded — and nothing reaches the feed — without this.
	if uc.operations != nil {
		uc.operations.Connect()
	}

	var batches <-chan livestream.Batch
	var id uint64
	if uc.stream != nil {
		id, batches = uc.stream.Subscribe()
	}

	return batches, func() {
		if uc.stream != nil {
			uc.stream.Unsubscribe(id)
		}

		if uc.operations != nil {
			uc.operations.Disconnect()
		}

		if uc.metrics != nil {
			uc.metrics.Disconnect()
		}
	}
}

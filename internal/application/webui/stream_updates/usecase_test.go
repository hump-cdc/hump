package streamupdates

import (
	"slices"
	"testing"

	livestream "hump/internal/port/live_stream"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"
)

func TestSubscribeAttachesAndReleasesTheClient(t *testing.T) {
	var log []string
	metrics := &fakeMetrics{log: &log}
	operations := &fakeOperations{log: &log}
	stream := &fakeStreamer{log: &log, batches: make(chan livestream.Batch)}

	batches, release := NewUseCase(metrics, operations, stream).Subscribe()

	if batches != (<-chan livestream.Batch)(stream.batches) {
		t.Error("Subscribe() returned a channel other than the feed's")
	}

	release()

	want := []string{
		"metrics connect", "operations connect", "subscribe",
		"unsubscribe 7", "operations disconnect", "metrics disconnect",
	}
	if !slices.Equal(log, want) {
		t.Errorf("calls = %q, want %q", log, want)
	}
}

func TestSubscribeWithEverythingDisabled(t *testing.T) {
	batches, release := NewUseCase(nil, nil, nil).Subscribe()

	if batches != nil {
		t.Error("Subscribe() returned a channel, want nil so the feed blocks forever")
	}

	release()
}

// fakeMetrics logs Connect and Disconnect; nothing else is reachable.
type fakeMetrics struct {
	metricscollector.Collector

	log *[]string
}

func (m *fakeMetrics) Connect()    { *m.log = append(*m.log, "metrics connect") }
func (m *fakeMetrics) Disconnect() { *m.log = append(*m.log, "metrics disconnect") }

// fakeOperations logs Connect and Disconnect; nothing else is reachable.
type fakeOperations struct {
	operationscollector.Collector

	log *[]string
}

func (o *fakeOperations) Connect()    { *o.log = append(*o.log, "operations connect") }
func (o *fakeOperations) Disconnect() { *o.log = append(*o.log, "operations disconnect") }

// fakeStreamer hands out batches under id 7 and logs every call.
type fakeStreamer struct {
	log     *[]string
	batches chan livestream.Batch
}

func (s *fakeStreamer) Subscribe() (uint64, <-chan livestream.Batch) {
	*s.log = append(*s.log, "subscribe")

	return 7, s.batches
}

func (s *fakeStreamer) Unsubscribe(id uint64) {
	*s.log = append(*s.log, "unsubscribe "+string(rune('0'+id)))
}

package streamupdates

import (
	"slices"
	"testing"

	livestream "hump/internal/port/live_stream"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"
	"hump/internal/port/snapshot"
)

func TestSubscribeAttachesAndReleasesTheClient(t *testing.T) {
	var log []string
	metrics := &fakeMetrics{log: &log}
	operations := &fakeOperations{log: &log}
	stream := &fakeStreamer{log: &log, batches: make(chan livestream.Batch)}
	runner := &fakeRunner{
		log:     &log,
		current: snapshot.State{Status: snapshot.StatusProcessing},
		updates: make(chan snapshot.State),
	}

	feed, release := NewUseCase(metrics, operations, stream, runner).Subscribe()

	if feed.Batches != (<-chan livestream.Batch)(stream.batches) {
		t.Error("Subscribe() returned a batch channel other than the feed's")
	}

	if feed.Snapshots != (<-chan snapshot.State)(runner.updates) {
		t.Error("Subscribe() returned a snapshot channel other than the runner's")
	}

	if feed.Snapshot.Status != snapshot.StatusProcessing {
		t.Errorf("Snapshot.Status = %q, want the runner's %q", feed.Snapshot.Status, snapshot.StatusProcessing)
	}

	release()

	want := []string{
		"metrics connect", "operations connect", "subscribe", "snapshot subscribe",
		"snapshot unsubscribe 3", "unsubscribe 7", "operations disconnect", "metrics disconnect",
	}
	if !slices.Equal(log, want) {
		t.Errorf("calls = %q, want %q", log, want)
	}
}

func TestSubscribeWithEverythingDisabled(t *testing.T) {
	feed, release := NewUseCase(nil, nil, nil, nil).Subscribe()

	if feed.Batches != nil {
		t.Error("Subscribe() returned a batch channel, want nil so the feed blocks forever")
	}

	if feed.Snapshots != nil {
		t.Error("Subscribe() returned a snapshot channel, want nil so it blocks forever")
	}

	if feed.Snapshot.Status != snapshot.StatusIdle {
		t.Errorf("Snapshot.Status = %q, want %q", feed.Snapshot.Status, snapshot.StatusIdle)
	}

	release()
}

func TestProgressSumsRowsAcrossTables(t *testing.T) {
	state := snapshot.State{Status: snapshot.StatusProcessing, Tables: []snapshot.Table{
		{CompletedRows: 50, EstimatedRows: 100},
		{CompletedRows: 49, EstimatedRows: 200},
	}}

	if got := progress(state); got != 33 {
		t.Errorf("progress() = %d, want 33", got)
	}
}

func TestProgressIsZeroWithoutAnEstimate(t *testing.T) {
	state := snapshot.State{Status: snapshot.StatusProcessing, Tables: []snapshot.Table{{}, {}}}

	if got := progress(state); got != 0 {
		t.Errorf("progress() = %d, want 0", got)
	}
}

func TestProgressIsZeroOutsideProcessing(t *testing.T) {
	state := snapshot.State{Status: snapshot.StatusDone, Tables: []snapshot.Table{
		{CompletedRows: 50, EstimatedRows: 100},
	}}

	if got := progress(state); got != 0 {
		t.Errorf("progress() = %d, want 0", got)
	}
}

func TestProgressNeverExceeds100(t *testing.T) {
	state := snapshot.State{Status: snapshot.StatusProcessing, Tables: []snapshot.Table{
		{CompletedRows: 300, EstimatedRows: 100},
	}}

	if got := progress(state); got != 100 {
		t.Errorf("progress() = %d, want 100", got)
	}
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

// fakeRunner hands out current and updates under id 3 and logs every call;
// nothing else is reachable.
type fakeRunner struct {
	snapshot.Runner

	log     *[]string
	current snapshot.State
	updates chan snapshot.State
}

func (r *fakeRunner) Subscribe() (uint64, snapshot.State, <-chan snapshot.State) {
	*r.log = append(*r.log, "snapshot subscribe")

	return 3, r.current, r.updates
}

func (r *fakeRunner) Unsubscribe(id uint64) {
	*r.log = append(*r.log, "snapshot unsubscribe "+string(rune('0'+id)))
}

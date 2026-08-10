package retry

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"hump/internal/telemetry/metrics"
	"hump/internal/types/config"
	"hump/internal/types/stream"
	"hump/pkg/limit"
)

type fakePublisher struct {
	mu     sync.Mutex
	events []stream.Message
}

func (f *fakePublisher) Publish(event stream.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakePublisher) HasSubscribers() bool { return true }

func (f *fakePublisher) snapshot() []stream.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]stream.Message(nil), f.events...)
}

func newRetry() *retry {
	m, _ := metrics.New("test")
	var wg sync.WaitGroup
	return New(&wg, "", limit.New(m), m)
}

func linearRetry(maxAttempts int, timeout time.Duration) *config.Retry {
	return &config.Retry{
		MaxAttempts:    maxAttempts,
		AttemptTimeout: timeout,
		Backoff:        config.Backoff{Type: config.BackoffTypeLinear},
	}
}

func TestRunWithRetryStreamsOperationOnSuccess(t *testing.T) {
	r := newRetry()
	pub := &fakePublisher{}
	base := stream.Event{ID: "op1", Table: "users"}

	handler := func(_ context.Context, evt *stream.Event) error {
		if evt != nil {
			evt.Payload = "p"
		}
		return nil
	}

	if err := r.runWithRetry(context.Background(), "kafka", "name", "host", "target", linearRetry(3, time.Second), base, pub, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := pub.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %+v", len(events), events)
	}
	op, ok := events[0].(stream.Event)
	if !ok {
		t.Fatalf("expected stream.Event, got %T", events[0])
	}
	if !op.Success || op.Error != "" || op.ID != "op1" || op.Payload != "p" {
		t.Fatalf("unexpected operation event: %+v", op)
	}
}

func TestRunWithRetryStreamsRetriesLinkedToOperation(t *testing.T) {
	r := newRetry()
	pub := &fakePublisher{}
	base := stream.Event{ID: "op1", Table: "users"}

	calls := 0
	handler := func(_ context.Context, evt *stream.Event) error {
		calls++
		if evt != nil {
			evt.Payload = "p"
		}
		if calls < 3 {
			return errors.New("boom")
		}
		return nil
	}

	if err := r.runWithRetry(context.Background(), "kafka", "name", "host", "target", linearRetry(3, time.Second), base, pub, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := pub.snapshot()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %+v", len(events), events)
	}

	op, ok := events[0].(stream.Event)
	if !ok || op.Success || op.Error != "boom" || op.ID != "op1" || op.Payload != "p" {
		t.Fatalf("unexpected operation event: %#v", events[0])
	}

	r2, ok := events[1].(stream.RetryEvent)
	if !ok || r2.OperationID != "op1" || r2.Attempt != 2 || r2.Success || r2.Error != "boom" {
		t.Fatalf("unexpected retry event 2: %#v", events[1])
	}

	r3, ok := events[2].(stream.RetryEvent)
	if !ok || r3.OperationID != "op1" || r3.Attempt != 3 || !r3.Success || r3.Error != "" {
		t.Fatalf("unexpected retry event 3: %#v", events[2])
	}
}

func TestRunWithRetryTimeoutDoesNotReadSharedEvent(t *testing.T) {
	r := newRetry()
	pub := &fakePublisher{}
	base := stream.Event{ID: "op1", Table: "users"}

	// The handler blocks past the attempt timeout, then writes the shared event
	// only after being released. The operation event must be published from the
	// private baseEvent copy (Payload empty), proving the timeout path never reads
	// the still-mutating shared event. Run with -race to catch a regression.
	release := make(chan struct{})
	defer close(release)

	handler := func(_ context.Context, evt *stream.Event) error {
		<-release
		if evt != nil {
			evt.Payload = "late"
		}
		return nil
	}

	err := r.runWithRetry(context.Background(), "kafka", "name", "host", "target", linearRetry(1, time.Millisecond), base, pub, handler)
	if err == nil {
		t.Fatal("expected a timeout error")
	}

	events := pub.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %+v", len(events), events)
	}
	op, ok := events[0].(stream.Event)
	if !ok || op.Success || op.Payload != "" || op.ID != "op1" {
		t.Fatalf("unexpected operation event: %#v", events[0])
	}
	if !strings.Contains(op.Error, "timeout") && !strings.Contains(op.Error, "deadline") {
		t.Fatalf("expected timeout error, got %q", op.Error)
	}
}

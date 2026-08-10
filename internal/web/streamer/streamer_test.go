package streamer

import (
	"testing"
	"time"

	"hump/internal/types/stream"
	"hump/internal/types/wal"
)

func recv(t *testing.T, ch <-chan interface{}) (interface{}, bool) {
	t.Helper()
	select {
	case ev, ok := <-ch:
		return ev, ok
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return nil, false
	}
}

func TestHasSubscribers(t *testing.T) {
	s := New[stream.Message]()

	if s.HasSubscribers() {
		t.Fatal("expected no subscribers on a fresh streamer")
	}

	id, _ := s.Subscribe()
	if !s.HasSubscribers() {
		t.Fatal("expected a subscriber after Subscribe")
	}

	s.Unsubscribe(id)
	if s.HasSubscribers() {
		t.Fatal("expected no subscribers after Unsubscribe")
	}
}

func TestFanOut(t *testing.T) {
	s := New[stream.Message]()

	_, a := s.Subscribe()
	_, b := s.Subscribe()

	want := stream.Event{Schema: "public", Table: "users", Operation: wal.WalOperationInsert}
	s.Publish(want)

	if got, ok := recv(t, a); !ok || got.(stream.Event).Table != want.Table {
		t.Fatalf("subscriber a: got %+v ok=%v", got, ok)
	}
	if got, ok := recv(t, b); !ok || got.(stream.Event).Table != want.Table {
		t.Fatalf("subscriber b: got %+v ok=%v", got, ok)
	}
}

func TestSlowSubscriberDropsWithoutBlocking(t *testing.T) {
	s := New[stream.Message]()

	id, ch := s.Subscribe()

	// Overflow the subscriber buffer plus a margin. Publish must not block even
	// though the subscriber never drains; excess events are dropped.
	done := make(chan struct{})
	go func() {
		for range subscriberBuffer * 3 {
			s.Publish(stream.Event{Table: "t"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full slow subscriber")
	}

	// The subscriber received at most its buffer's worth of events.
	count := 0
	for {
		select {
		case <-ch:
			count++
			continue
		default:
		}
		break
	}
	if count > subscriberBuffer {
		t.Fatalf("expected at most %d buffered events, got %d", subscriberBuffer, count)
	}

	s.Unsubscribe(id)
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	s := New[stream.Message]()

	id, ch := s.Subscribe()
	s.Unsubscribe(id)

	if _, ok := recv(t, ch); ok {
		t.Fatal("expected channel to be closed after Unsubscribe")
	}
}

func TestCloseClosesSubscriberChannels(t *testing.T) {
	s := New[stream.Message]()

	_, a := s.Subscribe()
	_, b := s.Subscribe()

	s.close()

	if _, ok := recv(t, a); ok {
		t.Fatal("subscriber a: expected channel closed after close")
	}
	if _, ok := recv(t, b); ok {
		t.Fatal("subscriber b: expected channel closed after close")
	}
}

func TestPublishAfterCloseIsNoOp(t *testing.T) {
	s := New[stream.Message]()
	s.close()

	// Must not panic on a send to the closed in channel, and must not block.
	done := make(chan struct{})
	go func() {
		s.Publish(stream.Event{Table: "t"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked after close")
	}
}

func TestSubscribeAfterCloseReturnsClosedChannel(t *testing.T) {
	s := New[stream.Message]()
	s.close()

	_, ch := s.Subscribe()
	if _, ok := recv(t, ch); ok {
		t.Fatal("expected a closed channel from Subscribe after close")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	s := New[stream.Message]()

	s.close()
	s.close() // must not panic
}

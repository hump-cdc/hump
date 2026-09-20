package idlegate

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestGateIdleUntilConnect(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)

	if g.Active() {
		t.Fatal("Active() = true, want false before Connect")
	}

	if starts, stops := calls.counts(); starts != 0 || stops != 0 {
		t.Fatalf("callbacks ran %d/%d times, want 0/0 before Connect", starts, stops)
	}
}

func TestGateActivatesOnFirstConnect(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Connect()

	if !g.Active() {
		t.Fatal("Active() = false, want true after Connect")
	}

	if starts, _ := calls.counts(); starts != 1 {
		t.Fatalf("onStart ran %d times, want 1", starts)
	}
}

func TestGateStartsOnceWhileClientsRemain(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Connect()
	g.Connect()

	if starts, _ := calls.counts(); starts != 1 {
		t.Fatalf("onStart ran %d times for two clients, want 1", starts)
	}
}

func TestGateStaysActiveWhileClientsRemain(t *testing.T) {
	g, calls := newTestGate(t, time.Millisecond)
	g.Connect()
	g.Connect()
	g.Disconnect()

	time.Sleep(20 * time.Millisecond)

	if !g.Active() {
		t.Fatal("Active() = false, want true while a client is still connected")
	}

	if _, stops := calls.counts(); stops != 0 {
		t.Fatalf("onStop ran %d times, want 0 while a client is still connected", stops)
	}
}

func TestGateDeactivatesAfterIdleTimeout(t *testing.T) {
	g, calls := newTestGate(t, time.Millisecond)
	g.Connect()
	g.Disconnect()

	waitFor(t, func() bool { return !g.Active() })

	if _, stops := calls.counts(); stops != 1 {
		t.Fatalf("onStop ran %d times, want 1 after the idle timeout", stops)
	}
}

func TestGateStopsCallbackBeforeClearingActive(t *testing.T) {
	var observed bool

	g := New(time.Millisecond, func() {}, func() { observed = true })
	t.Cleanup(g.Close)

	g.Connect()
	g.Disconnect()

	waitFor(t, func() bool { return !g.Active() })

	if !observed {
		t.Fatal("Active() reported false before onStop ran; a caller can observe torn-down state as live")
	}
}

func TestGateReconnectCancelsIdleTimeout(t *testing.T) {
	g, calls := newTestGate(t, 20*time.Millisecond)
	g.Connect()
	g.Disconnect()
	g.Connect()

	time.Sleep(60 * time.Millisecond)

	if !g.Active() {
		t.Fatal("Active() = false, want true after a reconnect")
	}

	if starts, stops := calls.counts(); starts != 1 || stops != 0 {
		t.Fatalf("callbacks ran %d/%d times, want 1/0 across a reconnect", starts, stops)
	}
}

func TestGateIgnoresStaleDeactivation(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Connect()

	g.deactivate(0)

	if !g.Active() {
		t.Fatal("Active() = false, want true after a deactivation of an older generation")
	}

	if _, stops := calls.counts(); stops != 0 {
		t.Fatalf("onStop ran %d times for a stale generation, want 0", stops)
	}
}

func TestGateIgnoresDisconnectWithoutConnect(t *testing.T) {
	g, _ := newTestGate(t, time.Minute)
	g.Disconnect()

	if g.connections != 0 {
		t.Fatalf("connections = %d, want 0", g.connections)
	}

	g.Connect()
	g.Close()
	g.Disconnect()

	if g.connections != 1 {
		t.Fatalf("connections = %d after a disconnect past close, want the count left alone", g.connections)
	}
}

func TestGateIgnoresConnectAfterClose(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Close()
	g.Connect()

	if g.Active() {
		t.Fatal("Active() = true, want false after Close")
	}

	if starts, _ := calls.counts(); starts != 0 {
		t.Fatalf("onStart ran %d times after Close, want 0", starts)
	}
}

func TestGateCloseStopsActiveGate(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Connect()
	g.Close()

	if g.Active() {
		t.Fatal("Active() = true, want false after Close")
	}

	if _, stops := calls.counts(); stops != 1 {
		t.Fatalf("onStop ran %d times, want 1", stops)
	}
}

func TestGateCloseIsIdempotent(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Connect()
	g.Close()
	g.Close()

	if _, stops := calls.counts(); stops != 1 {
		t.Fatalf("onStop ran %d times for two Close calls, want 1", stops)
	}
}

func TestGateSkipsStopOnCloseWhileIdle(t *testing.T) {
	g, calls := newTestGate(t, time.Minute)
	g.Close()

	if _, stops := calls.counts(); stops != 0 {
		t.Fatalf("onStop ran %d times on a gate that never activated, want 0", stops)
	}
}

func TestGateHasNoBackgroundGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t)

	g, _ := newTestGate(t, time.Millisecond)
	g.Connect()
	g.Disconnect()

	waitFor(t, func() bool { return !g.Active() })
}

// newTestGate returns a gate whose callbacks only count their calls, closed when
// the test ends.
func newTestGate(t *testing.T, timeout time.Duration) (*Gate, *callCounter) {
	t.Helper()

	calls := &callCounter{}
	g := New(timeout, calls.start, calls.stop)
	t.Cleanup(g.Close)

	return g, calls
}

// callCounter counts the gate's callbacks. It carries its own lock because the
// idle timer fires onStop on a goroutine of its own.
type callCounter struct {
	mu     sync.Mutex
	starts int
	stops  int
}

// start records an onStart call.
func (c *callCounter) start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.starts++
}

// stop records an onStop call.
func (c *callCounter) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stops++
}

// counts returns how often each callback has run.
func (c *callCounter) counts() (starts, stops int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.starts, c.stops
}

// waitFor polls cond for a second and fails the test if it never holds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition not met within a second")
}

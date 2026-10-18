// Package idlegate holds a resource active while clients are connected, and for
// an idle period after the last one leaves, so an unwatched resource costs nothing.
package idlegate

import (
	"sync"
	"sync/atomic"
	"time"
)

// Gate activates a resource on the first Connect and deactivates it once no
// client has been connected for its idle timeout. The zero value is not usable;
// call New.
//
// onStart and onStop run while the gate's lock is held, so neither may call back
// into the gate. A callback that takes a lock of its own establishes the order
// gate-then-callback, which every other path must respect.
type Gate struct {
	timeout time.Duration
	onStart func()
	onStop  func()

	active atomic.Bool // active mirrors the callbacks; Active reads it without the lock.

	mu          sync.Mutex
	connections int
	generation  uint64 // generation changes on every Connect, so an idle timer armed before it does not deactivate the gate.
	idle        *time.Timer
	closed      bool
}

// New returns an idle gate that calls onStart on the first Connect and onStop
// once timeout has passed without a client.
func New(timeout time.Duration, onStart, onStop func()) *Gate {
	return &Gate{
		timeout: timeout,
		onStart: onStart,
		onStop:  onStop,
	}
}

// Connect registers a client, cancels a pending deactivation and activates the
// gate if it was idle. It does nothing once the gate is closed.
func (g *Gate) Connect() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed {
		return
	}

	g.connections++
	g.generation++

	if g.idle != nil {
		g.idle.Stop()
		g.idle = nil
	}

	if !g.active.Load() {
		g.activate()
	}
}

// Disconnect unregisters a client and, once none is left, arms the idle timer.
// An unpaired call does nothing rather than driving the count negative.
func (g *Gate) Disconnect() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed || g.connections == 0 {
		return
	}

	g.connections--
	if g.connections > 0 {
		return
	}

	generation := g.generation
	g.idle = time.AfterFunc(g.timeout, func() { g.deactivate(uint64(generation)) })
}

// Active reports whether the gate is activated. It is a hint rather than a
// guarentee: the last client may leave right after the call returns.
func (g *Gate) Active() bool {
	return g.active.Load()
}

// Close deactivates the gate for good; a later Connect does nothing. It is
// idempotent.
func (g *Gate) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.closed = true
	g.stop()
}

// activate runs onStart and marks the gate active. g.mu must be held.
func (g *Gate) activate() {
	g.onStart()
	g.active.Store(true)
}

// deactivate stops the gate unless a client connected after the idle timer of
// generation was armed.
func (g *Gate) deactivate(generation uint64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed || g.generation != generation || g.connections > 0 {
		return
	}

	g.stop()
}

// stop runs onStop and drops the idle timer. The flag is cleared after the
// callback, so a caller that sees Active report false also sees whatever onStop
// tore down. g.mu must be held.
func (g *Gate) stop() {
	if g.active.Load() {
		g.onStop()
		g.active.Store(false)
	}

	if g.idle != nil {
		g.idle.Stop()
		g.idle = nil
	}
}

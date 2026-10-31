package postgres

import (
	"context"
	"sync"
	"time"

	"hump/internal/domain"
	"hump/internal/port/database"
)

// walGate pauses WAL processing at the handler: while paused, a decoded change
// waits before it is delivered. Pausing is a flag, not a counter, so any number
// of pauses is undone by one resume.
type walGate struct {
	mu      sync.Mutex
	paused  bool
	active  int           // active counts handler calls in flight.
	resumed chan struct{} // resumed is closed by resume; set while paused.
	drained chan struct{} // drained is closed once active drops to zero; set while pause waits.
}

// newWalGate returns an open gate.
func newWalGate() *walGate {
	return &walGate{}
}

// pause closes the gate and waits until no handler call is in flight, or
// returns ctx.Err(), leaving the gate closed either way.
func (g *walGate) pause(ctx context.Context) error {
	g.mu.Lock()
	if !g.paused {
		g.paused = true
		g.resumed = make(chan struct{})
	}

	if g.active == 0 {
		g.mu.Unlock()
		return nil
	}

	if g.drained == nil {
		g.drained = make(chan struct{})
	}
	drained := g.drained
	g.mu.Unlock()

	select {
	case <-drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// resume opens the gate, releasing every call waiting in enter. It is a no-op
// on an open gate.
func (g *walGate) resume() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.paused {
		g.paused = false
		close(g.resumed)
	}
}

// wrap returns handler gated by g. A call waits on stop rather than on its own
// context, which WAL hands down detached, so a shutdown still ends the wait.
func (g *walGate) wrap(stop context.Context, handler database.ProcessMessageHandler) database.ProcessMessageHandler {
	return func(ctx context.Context, timestamp time.Time, operation domain.OperationType, dbName domain.DbName, schema domain.TableSchema, table domain.TableName, headers domain.Headers, metadata database.MessageMetadata, prev domain.DbRecord, next domain.DbRecord) error {
		if err := g.enter(stop); err != nil {
			return err
		}
		defer g.leave()

		return handler(ctx, timestamp, operation, dbName, schema, table, headers, metadata, prev, next)
	}
}

// enter waits for the gate to open and registers a handler call in flight, or
// returns ctx.Err() without registering one.
func (g *walGate) enter(ctx context.Context) error {
	for {
		g.mu.Lock()
		if !g.paused {
			g.active++
			g.mu.Unlock()
			return nil
		}
		resumed := g.resumed
		g.mu.Unlock()

		select {
		case <-resumed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// leave unregisters a handler call registered by enter.
func (g *walGate) leave() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.active--
	if g.active == 0 && g.drained != nil {
		close(g.drained)
		g.drained = nil
	}
}

package shutdown

import (
	"context"
	"sync"
	"time"
)

// Stage orders shutdown handlers: every handler of a stage finishes before the
// next stage starts.
type Stage int

const (
	// StageWorkers stops workers processing messages.
	StageWorkers Stage = iota
	// StageSinks closes persistent sink's connections.
	StageSinks
	// StageServers stops the HTTP servers.
	StageServers

	stageCount
)

var (
	globalRegistry = &registry{}
	drainOnce      sync.Once
	draining       = make(chan struct{})
)

// OnShutdown registers a named handler to run during stage.
func OnShutdown(stage Stage, name string, fn func(context.Context) error) {
	globalRegistry.add(stage, name, fn)
}

// Shutdown runs all registered handlers stage by stage, each bounded by timeout.
func Shutdown(timeout time.Duration) error {
	return globalRegistry.shutdown(timeout)
}

// Drain signals that the process is shutting down: work already picked up is
// finished, but nothing new, retries included, is started. Safe to call twice.
func Drain() {
	drainOnce.Do(func() { close(draining) })
}

// Draining returns a channel that is closed once Drain has been called.
func Draining() <-chan struct{} {
	return draining
}

package shutdown

import (
	"context"
	"time"
)

var globalRegistry = &registry{}

// Registers a shutdown handler with the given name in the global shutdown registry.
func OnShutdown(name string, fn func(context.Context) error) {
	globalRegistry.add(name, fn)
}

// Shutdown runs all registered handlers in reverse order within the given timeout.
func Shutdown(timeout time.Duration) error {
	return globalRegistry.shutdown(timeout)
}

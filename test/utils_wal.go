package test

import (
	"context"
	"fmt"
	"hump/internal/telemetry/health"
	"time"
)

// waitWalReady makes sure the WAL sender is operational before proceeding with the test.
func waitWalReady(ctx context.Context, h *health.Health, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.WalSender.Operational() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("WAL sender did not become ready within %s", timeout)
}

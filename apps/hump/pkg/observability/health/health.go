// Package health exposes the health-check endpoint and the checks components
// flip as their connections come and go.
package health

import (
	"fmt"
	"hump/pkg/version"
	"net/http"

	hg "github.com/hellofresh/health-go/v5"
)

// Health is the health-check endpoint with one check per monitored component.
type Health struct {
	health *hg.Health

	Postgres  *Check
	WalSender *Check
}

// New creates the health checks for component, all starting unhealthy.
func New(component string) (*Health, error) {
	h := &Health{
		Postgres:  newCheck("Postgres"),
		WalSender: newCheck("WAL Sender"),
	}

	health, err := hg.New(
		hg.WithComponent(hg.Component{
			Name:    component,
			Version: version.Version,
		}),
		h.Postgres.configure(),
		h.WalSender.configure(),
	)

	if err != nil {
		return nil, fmt.Errorf("health: create health: %w", err)
	}

	h.health = health
	return h, nil
}

// Handler returns the HTTP handler that reports the checks.
func (h *Health) Handler() http.Handler {
	return h.health.Handler()
}

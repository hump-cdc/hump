package health

import (
	"fmt"
	"hump/pkg/version"
	"net/http"

	hg "github.com/hellofresh/health-go/v5"
)

type Health struct {
	health *hg.Health

	Postgres  *HealthCheck
	WalSender *HealthCheck
}

func New(component string) (*Health, error) {
	h := &Health{
		Postgres:  newHealthCheck("Postgres"),
		WalSender: newHealthCheck("WAL Sender"),
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

func (h *Health) Handler() http.Handler {
	return h.health.Handler()
}

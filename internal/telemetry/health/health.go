// Package health exposes the liveness, readiness, and startup probes that
// back the telemetry server's /health/* endpoints. Probe state is updated
// from the components they observe (Postgres connectivity and the WAL
// sender) and read on every HTTP request.
package health

import (
	"hump/pkg/version"
	"net/http"

	hg "github.com/hellofresh/health-go/v5"
)

// Health groups the individually updatable probes that feed the
// startup/ready/live HTTP handlers. Callers mutate each probe through its
// Update method as the corresponding subsystem changes state.
type Health struct {
	Postgres  *healthCheck
	WalSender *healthCheck
}

// New returns a [Health] together with the three HTTP handlers serving the
// startup, readiness, and liveness probes. The liveness handler tolerates a
// failing WAL sender to avoid restart loops during transient reconnects,
// while startup and readiness require every component to be operational.
func New() (health *Health, startupHandler http.Handler, readyHandler http.Handler, liveHandler http.Handler) {
	health = &Health{
		Postgres:  &healthCheck{name: "Postgres"},
		WalSender: &healthCheck{name: "WAL Sender"},
	}

	startup, _ := hg.New(
		hg.WithComponent(hg.Component{Name: "hump", Version: version.Version}),
		health.Postgres.check(false),
		health.WalSender.check(false),
	)
	startupHandler = startup.Handler()

	ready, _ := hg.New(
		hg.WithComponent(hg.Component{Name: "hump", Version: version.Version}),
		health.Postgres.check(false),
		health.WalSender.check(false),
	)
	readyHandler = ready.Handler()

	live, _ := hg.New(
		hg.WithComponent(hg.Component{Name: "hump", Version: version.Version}),
		health.Postgres.check(false),
		health.WalSender.check(true),
	)
	liveHandler = live.Handler()

	return
}

package web

import (
	"hump/internal/web/routes"

	"github.com/danielgtaylor/huma/v2"
)

func (w *Web) routes(api huma.API) {
	routes.RegisterConfig(api, w.opts, w.cfg)
	routes.RegisterMetrics(api, w.metricsHandler)
	routes.RegisterOperations(api, w.streamer)
}

package web

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

func registerRoutes(api huma.API, router chi.Router, metricsHandler http.Handler) {
	_ = api

	if metricsHandler != nil {
		router.Handle("/api/metrics", metricsHandler)
	}
}

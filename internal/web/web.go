// Package web exposes the HTTP server that serves the bundled web UI
// (a single-page application embedded under assets/) and the Huma-backed
// JSON API mounted under /api.
package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	"hump/internal/app/shutdown"
	"hump/internal/log"
	"hump/pkg/version"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var logger = log.Scope("web")

//go:embed assets
var assetsFS embed.FS

// Web is the HTTP-facing surface that serves the embedded web UI and the
// JSON API on a single port.
type Web interface {
	// Serve binds to the given TCP port and blocks until the registered
	// shutdown hook closes the underlying server. It returns nil on a clean
	// shutdown and the server error otherwise.
	Serve(port uint) error
}

type web struct{}

// New constructs a [Web] server. It takes no dependencies yet; API handler
// dependencies should be threaded in here as routes are added.
func New() Web {
	return &web{}
}

// Serve starts the web HTTP server on the given port and registers a shutdown
// hook that gracefully stops it when the application shuts down. Requests to
// /api/* are handled by Huma; everything else is served from the embedded SPA
// assets with a fallback to index.html for client-side routes.
func (w *web) Serve(port uint) error {
	router := chi.NewMux()

	router.Use(middleware.Compress(5))

	api := humachi.New(router, huma.DefaultConfig("hump", version.Version))
	registerRoutes(api)

	assets, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return err
	}
	router.NotFound(staticHandler(assets))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	shutdown.OnShutdown("web", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	logger.Info(fmt.Sprintf("Web UI started at http://localhost:%d", port))
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

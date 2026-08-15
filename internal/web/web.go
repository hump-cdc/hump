package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/http"

	"hump/config"
	"hump/internal/adapter/postgres"
	"hump/internal/types/stream"
	"hump/internal/web/middleware"
	"hump/internal/web/streamer"
	"hump/pkg/app/options"
	"hump/pkg/app/shutdown"
	"hump/pkg/telemetry/log"
	"hump/pkg/version"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

var logger = log.Scope("web")

//go:embed assets
var assetsFS embed.FS

type Web struct {
	opts           options.Options
	cfg            config.Configuration
	db             *postgres.Postgres
	streamer       *streamer.FanOutStreamer[stream.Message]
	metricsHandler http.Handler
}

// New constructs a Web UI server.
func New(opts options.Options, cfg config.Configuration, db *postgres.Postgres, streamer *streamer.FanOutStreamer[stream.Message], metricsHandler http.Handler) *Web {
	return &Web{
		opts:           opts,
		cfg:            cfg,
		db:             db,
		streamer:       streamer,
		metricsHandler: metricsHandler,
	}
}

func (w *Web) Router() *chi.Mux {
	// Router
	router := chi.NewMux()

	// Middleware
	router.Use(chimiddleware.Compress(5))
	router.Use(middleware.Cors())
	router.NotFound(middleware.Static(assetsFS))

	return router
}

func (w *Web) Api(router *chi.Mux) (huma.API, error) {
	cfg := huma.DefaultConfig("hump", version.Version)
	cfg.CreateHooks = nil
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""

	api := humachi.New(router, cfg)
	w.routes(api)

	return api, nil
}

func (w *Web) Serve(ctx context.Context, port uint) error {
	router := w.Router()
	_, err := w.Api(router)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     router,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	shutdown.OnShutdown("web", func(shutdownCtx context.Context) error {
		return server.Shutdown(shutdownCtx)
	})

	logger.Info(fmt.Sprintf("Web UI started at http://localhost:%d", port))
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

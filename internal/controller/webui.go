package controller

import (
	"context"
	"embed"
	"errors"
	"fmt"
	getwebuiconfig "hump/internal/application/webui/get_webui_config"
	getwebuimetrics "hump/internal/application/webui/get_webui_metrics"
	getwebuioperations "hump/internal/application/webui/get_webui_operations"
	streamwebuiupdates "hump/internal/application/webui/stream_webui_updates"
	"hump/pkg/app/shutdown"
	"hump/pkg/middleware"
	"hump/pkg/telemetry/log"
	"hump/pkg/version"
	"net"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

//go:embed webui
var webuiFS embed.FS

type WebUIController struct {
	configController     *getwebuiconfig.Controller
	metricsController    *getwebuimetrics.Controller
	operationsController *getwebuioperations.Controller
	liveController       *streamwebuiupdates.Controller
}

func NewWebUIController(
	configController *getwebuiconfig.Controller,
	metricsController *getwebuimetrics.Controller,
	operationsController *getwebuioperations.Controller,
	liveController *streamwebuiupdates.Controller,
) *WebUIController {
	return &WebUIController{
		configController:     configController,
		metricsController:    metricsController,
		operationsController: operationsController,
		liveController:       liveController,
	}
}

func (c *WebUIController) Serve(ctx context.Context, port uint) error {
	logger := log.Scope("controller")

	// Router
	router := chi.NewMux()

	// Middleware
	router.Use(chimiddleware.Compress(5))
	router.Use(middleware.Cors())
	router.NotFound(middleware.Static(webuiFS, "webui"))

	// Routes
	c.Api(router)

	// Server
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     router,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Shutdown
	shutdown.OnShutdown(shutdown.StageServers, "web", func(shutdownCtx context.Context) error {
		return server.Shutdown(shutdownCtx)
	})

	// Listen
	logger.Info(fmt.Sprintf("Web UI started at http://localhost:%d", port))
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Api registers the web UI API routes on router and returns the resulting API,
// whose OpenAPI spec is also used to generate the web UI client.
func (c *WebUIController) Api(router chi.Router) huma.API {
	apiCfg := huma.DefaultConfig("hump", version.Version)
	apiCfg.CreateHooks = nil
	apiCfg.OpenAPIPath = ""
	apiCfg.DocsPath = ""
	apiCfg.SchemasPath = ""

	api := humachi.New(router, apiCfg)
	c.routes(api)

	return api
}

func (c *WebUIController) routes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "config.get",
		Method:      http.MethodGet,
		Path:        "/api/config",
		Summary:     "Get server configuration",
		Tags:        []string{"Config"},
	}, c.configController.Handler())

	huma.Register(api, huma.Operation{
		OperationID: "metrics.get",
		Method:      http.MethodPost,
		Path:        "/api/metrics",
		Summary:     "Get dashboard metrics aggregated by duration",
		Tags:        []string{"Metrics"},
	}, c.metricsController.Handler())

	huma.Register(api, huma.Operation{
		OperationID: "operations.list",
		Method:      http.MethodPost,
		Path:        "/api/operations",
		Summary:     "List sink deliveries matching a filter",
		Tags:        []string{"Operations"},
	}, c.operationsController.Handler())

	sse.Register(api, huma.Operation{
		OperationID: "live.stream",
		Method:      http.MethodGet,
		Path:        "/api/live",
		Summary:     "Stream live updates",
		Tags:        []string{"Live"},
	}, c.liveController.Events(), c.liveController.Handler())
}

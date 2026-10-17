package controller

import (
	"context"
	"embed"
	"errors"
	"fmt"
	getconfig "hump/internal/application/webui/get_config"
	getmetrics "hump/internal/application/webui/get_metrics"
	getoperations "hump/internal/application/webui/get_operations"
	getsnapshotestimates "hump/internal/application/webui/get_snapshot_estimates"
	getsnapshottables "hump/internal/application/webui/get_snapshot_tables"
	startsnapshot "hump/internal/application/webui/start_snapshot"
	stopsnapshot "hump/internal/application/webui/stop_snapshot"
	streamsnapshot "hump/internal/application/webui/stream_snapshot"
	streamupdates "hump/internal/application/webui/stream_updates"
	"hump/pkg/app/shutdown"
	"hump/pkg/middleware"
	"hump/pkg/observability/log"
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

// WebUIController serves the web UI API and the embedded SPA on one port.
type WebUIController struct {
	configController     *getconfig.Controller
	metricsController    *getmetrics.Controller
	operationsController *getoperations.Controller
	liveController       *streamupdates.Controller
	estimatesController  *getsnapshotestimates.Controller
	tablesController     *getsnapshottables.Controller
	snapshotController   *streamsnapshot.Controller
	startController      *startsnapshot.Controller
	stopController       *stopsnapshot.Controller
}

// NewWebUIController returns a WebUIController routing the API to the given
// per-use-case controllers.
func NewWebUIController(
	configController *getconfig.Controller,
	metricsController *getmetrics.Controller,
	operationsController *getoperations.Controller,
	liveController *streamupdates.Controller,
	estimatesController *getsnapshotestimates.Controller,
	tablesController *getsnapshottables.Controller,
	snapshotController *streamsnapshot.Controller,
	startController *startsnapshot.Controller,
	stopController *stopsnapshot.Controller,
) *WebUIController {
	return &WebUIController{
		configController:     configController,
		metricsController:    metricsController,
		operationsController: operationsController,
		liveController:       liveController,
		estimatesController:  estimatesController,
		tablesController:     tablesController,
		snapshotController:   snapshotController,
		startController:      startController,
		stopController:       stopController,
	}
}

// Serve listens on port until the registered shutdown hook closes the server.
// Requests inherit ctx; it returns nil on a clean shutdown.
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
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: readHeaderTimeout,
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

	huma.Register(api, huma.Operation{
		OperationID: "snapshot.tables",
		Method:      http.MethodPost,
		Path:        "/api/snapshot/tables",
		Summary:     "Describe configured tables with a suggested snapshot filter",
		Tags:        []string{"Snapshot"},
	}, c.tablesController.Handler())

	sse.Register(api, huma.Operation{
		OperationID: "snapshot.estimate",
		Method:      http.MethodPost,
		Path:        "/api/snapshot/estimate",
		Summary:     "Stream row count estimates of tables",
		Tags:        []string{"Snapshot"},
	}, c.estimatesController.Events(), c.estimatesController.Handler())

	sse.Register(api, huma.Operation{
		OperationID: "snapshot.stream",
		Method:      http.MethodGet,
		Path:        "/api/snapshot",
		Summary:     "Stream the state of the snapshot run",
		Tags:        []string{"Snapshot"},
	}, c.snapshotController.Events(), c.snapshotController.Handler())

	huma.Register(api, huma.Operation{
		OperationID:   "snapshot.start",
		Method:        http.MethodPost,
		Path:          "/api/snapshot/start",
		Summary:       "Start a snapshot of tables",
		Tags:          []string{"Snapshot"},
		DefaultStatus: http.StatusAccepted,
	}, c.startController.Handler())

	huma.Register(api, huma.Operation{
		OperationID:   "snapshot.stop",
		Method:        http.MethodPost,
		Path:          "/api/snapshot/stop",
		Summary:       "Stop the snapshot in progress, or clear a finished one",
		Tags:          []string{"Snapshot"},
		DefaultStatus: http.StatusNoContent,
	}, c.stopController.Handler())
}

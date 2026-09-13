package controller

import (
	"context"
	"embed"
	"errors"
	"fmt"
	getconfig "hump/internal/application/get_config"
	"hump/pkg/app/shutdown"
	"hump/pkg/middleware"
	"hump/pkg/telemetry/log"
	"hump/pkg/version"
	"net"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

//go:embed webui
var webuiFS embed.FS

type WebUIController struct {
	configController *getconfig.Controller
}

func NewWebUIController(configController *getconfig.Controller) *WebUIController {
	return &WebUIController{
		configController: configController,
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

	// huma.Register(api, huma.Operation{
	// 	OperationID: "metrics.get",
	// 	Method:      http.MethodGet,
	// 	Path:        "/api/metrics",
	// 	Summary:     "Get Prometheus metrics",
	// 	Tags:        []string{"Metrics"},
	// 	Responses: map[string]*huma.Response{
	// 		"200": {
	// 			Description: "Prometheus metrics in text exposition format",
	// 			Content: map[string]*huma.MediaType{
	// 				"text/plain": {Schema: &huma.Schema{Type: huma.TypeString}},
	// 			},
	// 		},
	// 	},
	// }, c.metricsController.Handler())

	// sse.Register(api, huma.Operation{
	// 	OperationID: "operations.stream",
	// 	Method:      http.MethodGet,
	// 	Path:        "/api/operations",
	// 	Summary:     "Stream change-data-capture operations",
	// 	Tags:        []string{"Operations"},
	// }, map[string]any{
	// 	"operation": stream.Event{},
	// 	"retry":     stream.RetryEvent{},
	// }, func(ctx context.Context, _ *struct{}, send sse.Sender) {
	// 	// s *streamer.FanOutStreamer[stream.Message]
	// 	if s == nil {
	// 		return
	// 	}

	// 	id, ch := s.Subscribe()
	// 	defer s.Unsubscribe(id)

	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case event, ok := <-ch:
	// 			if !ok {
	// 				return
	// 			}
	// 			if err := send(sse.Message{Data: event}); err != nil {
	// 				return
	// 			}
	// 		}
	// 	}
	// })
}

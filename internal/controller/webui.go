package controller

import (
	"context"
	"embed"
	"errors"
	"fmt"
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
var assetsFS embed.FS

// type Controller interface {
// 	Handler() func(ctx context.Context, _ *struct{}) (*huma.StreamResponse, error)
// }

type WebUIController struct {
	// configController  Controller
	// metricsController Controller
}

func NewWebUIController( /*configController Controller, metricsController Controller*/ ) *WebUIController {
	return &WebUIController{
		// configController:  configController,
		// metricsController: metricsController,
	}
}

func (c *WebUIController) Serve(ctx context.Context, port uint) error {
	logger := log.Scope("controller")

	// Router
	router := chi.NewMux()

	// Middleware
	router.Use(chimiddleware.Compress(5))
	router.Use(middleware.Cors())
	router.NotFound(middleware.Static(assetsFS))

	// Routes
	apiCfg := huma.DefaultConfig("hump", version.Version)
	apiCfg.CreateHooks = nil
	apiCfg.OpenAPIPath = ""
	apiCfg.DocsPath = ""
	apiCfg.SchemasPath = ""

	api := humachi.New(router, apiCfg)
	c.routes(api)

	// Server
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     router,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Shutdown
	shutdown.OnShutdown("web", func(shutdownCtx context.Context) error {
		return server.Shutdown(shutdownCtx)
	})

	// Listen
	logger.Info(fmt.Sprintf("Web UI started at http://localhost:%d", port))
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (c *WebUIController) routes(api huma.API) {
	// TODO: add routes

	// huma.Register(api, huma.Operation{
	// 	OperationID: "config.get",
	// 	Method:      http.MethodGet,
	// 	Path:        "/api/config",
	// 	Summary:     "Get server configuration",
	// 	Tags:        []string{"Config"},
	// }, c.configController.Handler())

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

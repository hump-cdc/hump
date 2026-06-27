package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterMetrics registers the Prometheus metrics endpoint on the huma API so
// it appears in the OpenAPI spec. The endpoint streams the Prometheus text
// exposition format produced by handler, bridging it through huma. The
// operation is registered even when handler is nil so the generated spec always
// includes it; in that case the endpoint reports 503.
func RegisterMetrics(api huma.API, handler http.Handler) {
	huma.Register(api, huma.Operation{
		OperationID: "metrics.get",
		Method:      http.MethodGet,
		Path:        "/api/metrics",
		Summary:     "Get Prometheus metrics",
		Tags:        []string{"Metrics"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Prometheus metrics in text exposition format",
				Content: map[string]*huma.MediaType{
					"text/plain": {Schema: &huma.Schema{Type: huma.TypeString}},
				},
			},
		},
	}, metricsHandler(handler))
}

func metricsHandler(handler http.Handler) func(ctx context.Context, _ *struct{}) (*huma.StreamResponse, error) {
	return func(_ context.Context, _ *struct{}) (*huma.StreamResponse, error) {
		if handler == nil {
			return nil, huma.Error503ServiceUnavailable("metrics disabled")
		}

		return &huma.StreamResponse{
			Body: func(hctx huma.Context) {
				req, _ := http.NewRequestWithContext(hctx.Context(), hctx.Method(), "/api/metrics", nil)
				hctx.EachHeader(func(name, value string) {
					req.Header.Add(name, value)
				})

				handler.ServeHTTP(&metricsResponseWriter{ctx: hctx}, req)
			},
		}, nil
	}
}

// metricsResponseWriter adapts huma.Context to http.ResponseWriter so a plain
// http.Handler (the Prometheus promhttp handler) can write its response through
// huma's streaming response.
type metricsResponseWriter struct {
	ctx     huma.Context
	header  http.Header
	written bool
}

func (w *metricsResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *metricsResponseWriter) WriteHeader(status int) {
	if w.written {
		return
	}
	w.written = true

	for name, values := range w.header {
		for i, value := range values {
			if i == 0 {
				w.ctx.SetHeader(name, value)
			} else {
				w.ctx.AppendHeader(name, value)
			}
		}
	}

	w.ctx.SetStatus(status)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ctx.BodyWriter().Write(b)
}

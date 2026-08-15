package getwebuimetrics

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type Controller struct {
	metricsHandler http.Handler
}

func NewController(metricsHandler http.Handler) *Controller {
	return &Controller{
		metricsHandler: metricsHandler,
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}) (*huma.StreamResponse, error) {
	return func(_ context.Context, _ *struct{}) (*huma.StreamResponse, error) {
		if c.metricsHandler == nil {
			return nil, huma.Error503ServiceUnavailable("metrics disabled")
		}

		return &huma.StreamResponse{
			Body: func(hctx huma.Context) {
				req, _ := http.NewRequestWithContext(hctx.Context(), hctx.Method(), "/api/metrics", nil)
				hctx.EachHeader(func(name, value string) {
					req.Header.Add(name, value)
				})

				c.metricsHandler.ServeHTTP(&metricsResponseWriter{ctx: hctx}, req)
			},
		}, nil
	}
}

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

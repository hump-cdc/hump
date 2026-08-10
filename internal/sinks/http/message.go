package http

import (
	"fmt"
	nethttp "net/http"

	"hump/internal/domain"
	"hump/internal/types/config"
	"hump/internal/types/sink"
	"hump/pkg/version"
)

type message struct {
	url    string
	method string
	body   string
}

// buildMessage renders the URL, method, and body templates against
// operationContext and returns the resulting [message].
func buildMessage(config config.HTTPSink, operationContext *sink.SinkContext) (*message, error) {
	url, err := config.URL.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render url: %w", err)
	}

	method, err := config.Method.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render method: %w", err)
	}

	body, err := config.Body.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render body: %w", err)
	}

	return &message{
		url:    url,
		method: method,
		body:   body,
	}, nil
}

// requestHeaders flattens the request headers into a plain string map (first
// value per key) for the operations stream. It returns nil when there are none.
func requestHeaders(header nethttp.Header) domain.Headers {
	if len(header) == 0 {
		return nil
	}

	out := make(domain.Headers, len(header))
	for name := range header {
		out[name] = header.Get(name)
	}
	return out
}

// fillHeaders sets the User-Agent to "hump/<version>" and applies each
// configured header template against operationContext.
func fillHeaders(req *nethttp.Request, config config.HTTPSink, operationContext *sink.SinkContext) error {
	req.Header.Set("User-Agent", "hump/"+version.Version)

	for name, tmpl := range config.Headers {
		v, err := tmpl.Render(operationContext)
		if err != nil {
			return fmt.Errorf("render header %s: %w", name, err)
		}
		if len(v) == 0 {
			continue
		}

		req.Header.Set(name, v)
	}

	return nil
}

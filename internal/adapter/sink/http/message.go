package http

import (
	"fmt"
	nethttp "net/http"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/version"
)

type message struct {
	url    string
	method string
	body   string
}

// buildMessage renders the URL, method, and body templates against
// operationContext and returns the resulting [message].
func buildMessage(config sinkcfg.HTTPSink, operationContext *sink.SinkContext) (*message, error) {
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

// requestHeaders records the request headers on capture, one entry per key
// holding the first value, prefixed so they do not collide with the sink's own
// metadata.
func requestHeaders(header nethttp.Header, capture *sink.Capture) {
	for name := range header {
		capture.Set("header."+name, header.Get(name))
	}
}

// fillHeaders sets the User-Agent to "hump/<version>" and applies each
// configured header template against operationContext.
func fillHeaders(req *nethttp.Request, config sinkcfg.HTTPSink, operationContext *sink.SinkContext) error {
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

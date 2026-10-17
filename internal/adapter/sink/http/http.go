// Package http implements the HTTP sink for hump.
package http

import (
	"bytes"
	"context"
	"fmt"
	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"io"
	nethttp "net/http"
	"strconv"
)

var defaultClient = &nethttp.Client{}

// Process delivers a single WAL operation to an HTTP sink by rendering the
// configured URL, method, body, and headers from operationContext and issuing
// the request with ctx. When capture is non-nil the request and the response
// status are recorded on it for the operations log.
func Process(ctx context.Context, config sinkcfg.HTTPSink, operationContext *sink.Context, capture *sink.Capture) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	message, err := buildMessage(config, operationContext)
	if err != nil {
		return wrapError(err)
	}

	req, err := nethttp.NewRequestWithContext(ctx, message.method, message.url, bytes.NewReader([]byte(message.body)))
	if err != nil {
		return wrapError(fmt.Errorf("build request for %s: %w: %w", message.url, errBuildRequest, err))
	}

	if err := fillHeaders(req, config, operationContext); err != nil {
		return wrapError(err)
	}

	resp, err := defaultClient.Do(req)

	// The capture is filled on both paths: a delivery that failed is the one a
	// reader most wants the request for.
	if capture != nil {
		capture.Target = message.url
		capture.SetPayload([]byte(message.body))
		requestHeaders(req.Header, capture)
		capture.Set("method", message.method)

		if resp != nil {
			capture.Set("status", strconv.Itoa(resp.StatusCode))
		}
	}

	if err != nil {
		return wrapError(fmt.Errorf("http request to %s: %w", message.url, err))
	}
	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return wrapError(&statusError{code: resp.StatusCode, url: message.url})
}

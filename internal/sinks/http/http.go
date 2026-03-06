package http

import (
	"bytes"
	"context"
	"fmt"
	"hump/internal/types/config"
	"hump/internal/types/sink"
	"io"
	nethttp "net/http"
)

var defaultClient = &nethttp.Client{}

// Process delivers a single WAL operation to an HTTP sink by rendering the
// configured URL, method, body, and headers from operationContext and issuing
// the request with ctx.
func Process(ctx context.Context, config config.HTTPSink, operationContext *sink.SinkContext) error {
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
	if err != nil {
		return wrapError(fmt.Errorf("http request to %s: %w", message.url, err))
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return wrapError(&statusError{code: resp.StatusCode, url: message.url})
}

package http

import (
	"errors"
	"fmt"
	"net"
	nethttp "net/http"

	"hump/internal/port/sink"
	"hump/pkg/template"
)

// errBuildRequest marks a failure to build the outgoing request from the
// rendered method, URL, and body. It stems from configuration and cannot
// succeed on retry.
var errBuildRequest = errors.New("build request")

// statusError is a non-2xx HTTP response. It carries the status code so the
// permanent/transient classification can inspect it.
type statusError struct {
	code int
	url  string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("http %d from %s", e.code, e.url)
}

// wrapError returns an "http" wrapped error, joining [sink.ErrSinkPermanent]
// for non-retriable errors: template render failures (shared across sinks) and
// the HTTP-specific failures classified by isPermanentHttpError.
func wrapError(err error) error {
	if errors.Is(err, template.ErrRender) || isPermanentHttpError(err) {
		return fmt.Errorf("http: %w", errors.Join(err, sink.ErrSinkPermanent))
	}
	return fmt.Errorf("http: %w", err)
}

// isPermanentHttpError reports whether an HTTP-specific error should bypass
// retries: a request that could not be built, a DNS name that does not exist,
// or a permanent response status (see isPermanentStatus).
func isPermanentHttpError(err error) bool {
	if errors.Is(err, errBuildRequest) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return true
	}

	if statusErr, ok := errors.AsType[*statusError](err); ok {
		return isPermanentStatus(statusErr.code)
	}

	return false
}

// isPermanentStatus reports whether an HTTP status code should be treated as
// a permanent (non-retriable) failure. All 4xx are permanent except 408
// (Request Timeout) and 429 (Too Many Requests), which remain retriable; 5xx
// and informational/success/redirect codes are not classified as permanent.
func isPermanentStatus(code int) bool {
	if code < 400 || code >= 500 {
		return false
	}
	switch code {
	case nethttp.StatusRequestTimeout, nethttp.StatusTooManyRequests:
		return false
	}
	return true
}

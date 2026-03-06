package config

import (
	_ "embed"
	"fmt"

	"hump/internal/template"
	"hump/internal/types/sink"

	"github.com/goccy/go-yaml/ast"
)

var (
	//go:embed templ/payload.tmpl
	defaultHttpPayload string

	//go:embed templ/client_id.tmpl
	defaultHttpClientID string

	//go:embed templ/idempotency_header.tmpl
	defaultHttpIdempotencyKey string

	//go:embed templ/traceparent_header.tmpl
	defaultHttpTraceparent string
)

type HTTPSink struct {
	URL *template.Template `yaml:"url"               validate:"required"`

	Method  *template.Template            `yaml:"method,omitempty"`
	Body    *template.Template            `yaml:"body,omitempty"`
	Headers map[string]*template.Template `yaml:"headers,omitempty"`
}

func (*HTTPSink) Type() sink.SinkType { return sink.SinkTypeHTTP }

func (s *HTTPSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sink.SinkTypeHTTP)
	}

	return fmt.Sprintf("%s(%s)", string(sink.SinkTypeHTTP), s.URL.Source())
}

// CrossValidate is a no-op for HTTP sinks; they have no cross-sink invariants.
func (*HTTPSink) CrossValidate([]Tables) error { return nil }

func (h *HTTPSink) UnmarshalYAML(node ast.Node) error {
	type http HTTPSink
	if err := decode(node, (*http)(h)); err != nil {
		return err
	}

	h.setDefaults()

	return nil
}

func (h *HTTPSink) setDefaults() {
	if h.Method == nil {
		t, _ := template.New("POST")
		h.Method = t
	}

	if h.Body == nil {
		t, _ := template.New(defaultHttpPayload)
		h.Body = t
	}

	if len(h.Headers) == 0 {
		h.Headers = map[string]*template.Template{}

		contentType, _ := template.New("application/json")
		h.Headers["Content-Type"] = contentType

		clientID, _ := template.New(defaultHttpClientID)
		h.Headers["X-Client-ID"] = clientID

		idempotency, _ := template.New(defaultHttpIdempotencyKey)
		h.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(defaultHttpTraceparent)
		h.Headers["traceparent"] = traceparent
	}
}

package sink

import (
	_ "embed"
	"fmt"

	"hump/config/decoder"
	sinkport "hump/internal/port/sink"
	"hump/pkg/metadata"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

type HTTPSink struct {
	URL *template.Template `yaml:"url"               validate:"required"`

	Method  *template.Template            `yaml:"method,omitempty"`
	Body    *template.Template            `yaml:"body,omitempty"`
	Headers map[string]*template.Template `yaml:"headers,omitempty"`
}

func (*HTTPSink) Type() sinkport.SinkType { return sinkport.SinkTypeHTTP }

func (s *HTTPSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeHTTP)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeHTTP), s.URL.Source())
}

func (s *HTTPSink) Metadata(context sinkport.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)

	return metadata.ParseHost(url), url
}

func (*HTTPSink) CrossValidate([][]sinkport.Definition) error { return nil }

func (h *HTTPSink) UnmarshalYAML(node ast.Node) error {
	type http HTTPSink
	if err := decoder.Decode(node, (*http)(h)); err != nil {
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
		t, _ := template.New(DefaultPayload)
		h.Body = t
	}

	if len(h.Headers) == 0 {
		h.Headers = map[string]*template.Template{}

		contentType, _ := template.New("application/json")
		h.Headers["Content-Type"] = contentType

		clientID, _ := template.New(DefaultClientID)
		h.Headers["X-Client-ID"] = clientID

		idempotency, _ := template.New(DefaultIdempotencyKey)
		h.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(DefaultTraceparent)
		h.Headers["traceparent"] = traceparent
	}
}

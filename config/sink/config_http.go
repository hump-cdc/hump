package sink

import (
	"fmt"

	"hump/config/decoder"
	sinkport "hump/internal/port/sink"
	"hump/pkg/metadata"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

// HTTPSink delivers each change as an HTTP request.
type HTTPSink struct {
	URL *template.Template `yaml:"url"               validate:"required"`

	Method  *template.Template            `yaml:"method,omitempty"`
	Body    *template.Template            `yaml:"body,omitempty"`
	Headers map[string]*template.Template `yaml:"headers,omitempty"`
}

// Type returns the sink type key.
func (*HTTPSink) Type() sinkport.Type { return sinkport.SinkTypeHTTP }

// DisplayName returns the sink type followed by its URL template.
func (s *HTTPSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sinkport.SinkTypeHTTP)
	}

	return fmt.Sprintf("%s(%s)", string(sinkport.SinkTypeHTTP), s.URL.Source())
}

// Metadata returns the host of the rendered URL and the URL itself.
func (s *HTTPSink) Metadata(context sinkport.Context) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)

	return metadata.ParseHost(url), url
}

// CrossValidate accepts any configuration: HTTP sinks share no state across tables.
func (*HTTPSink) CrossValidate([][]sinkport.Definition) error { return nil }

// UnmarshalYAML decodes the sink strictly and fills in its defaults.
func (s *HTTPSink) UnmarshalYAML(node ast.Node) error {
	type http HTTPSink
	if err := decoder.Decode(node, (*http)(s)); err != nil {
		return err
	}

	s.setDefaults()

	return nil
}

func (s *HTTPSink) setDefaults() {
	if s.Method == nil {
		t, _ := template.New("POST")
		s.Method = t
	}

	if s.Body == nil {
		t, _ := template.New(DefaultPayload)
		s.Body = t
	}

	if len(s.Headers) == 0 {
		s.Headers = map[string]*template.Template{}

		contentType, _ := template.New("application/json")
		s.Headers["Content-Type"] = contentType

		clientID, _ := template.New(DefaultClientID)
		s.Headers["X-Client-ID"] = clientID

		idempotency, _ := template.New(DefaultIdempotencyKey)
		s.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(DefaultTraceparent)
		s.Headers["traceparent"] = traceparent
	}
}

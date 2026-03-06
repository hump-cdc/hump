package config

import (
	_ "embed"

	"hump/internal/template"
)

var (
	//go:embed templ/payload.tmpl
	defaultRabbitMQPayload string

	//go:embed templ/idempotency_header.tmpl
	defaultRabbitMQIdempotencyKey string

	//go:embed templ/traceparent_header.tmpl
	defaultRabbitMQTraceparent string
)

// RabbitMQCommon holds the connection and message fields shared by the classic
// AMQP and RabbitMQ Streams sinks. It is embedded inline into both sink types.
type RabbitMQCommon struct {
	URL *template.Template `yaml:"url" validate:"required"`

	Payload *template.Template            `yaml:"payload,omitempty"`
	Headers map[string]*template.Template `yaml:"headers,omitempty"`
}

// setDefaults fills in the default payload and headers when unset.
func (c *RabbitMQCommon) setDefaults() {
	if c.Payload == nil {
		t, _ := template.New(defaultRabbitMQPayload)
		c.Payload = t
	}

	if len(c.Headers) == 0 {
		c.Headers = map[string]*template.Template{}

		idempotency, _ := template.New(defaultRabbitMQIdempotencyKey)
		c.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(defaultRabbitMQTraceparent)
		c.Headers["traceparent"] = traceparent
	}
}

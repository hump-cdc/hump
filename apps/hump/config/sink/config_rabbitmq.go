package sink

import (
	"hump/pkg/template"
)

// RabbitMQCommon is the connection and message settings shared by the AMQP and
// RabbitMQ Streams sinks.
type RabbitMQCommon struct {
	URL *template.Template `yaml:"url" validate:"required"`

	Payload *template.Template            `yaml:"payload,omitempty"`
	Headers map[string]*template.Template `yaml:"headers,omitempty"`
}

// SetDefaults fills in the default payload and the idempotency and traceparent
// headers when none are configured.
func (c *RabbitMQCommon) SetDefaults() {
	if c.Payload == nil {
		t, _ := template.New(DefaultPayload)
		c.Payload = t
	}

	if len(c.Headers) == 0 {
		c.Headers = map[string]*template.Template{}

		idempotency, _ := template.New(DefaultIdempotencyKey)
		c.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(DefaultTraceparent)
		c.Headers["traceparent"] = traceparent
	}
}

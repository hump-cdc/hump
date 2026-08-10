package config

import (
	_ "embed"

	"hump/pkg/template"

	"github.com/go-playground/validator/v10"
)

var (
	//go:embed templ/nats_subject.tmpl
	defaultNatsSubject string

	//go:embed templ/payload.tmpl
	defaultNatsPayload string

	//go:embed templ/idempotency_header.tmpl
	defaultNatsIdempotencyKey string

	//go:embed templ/traceparent_header.tmpl
	defaultNatsTraceparent string
)

// NatsCommon holds the connection, auth, and message fields shared by the Core
// NATS and JetStream sinks. It is embedded inline into both sink types.
type NatsCommon struct {
	URL  *template.Template `yaml:"url"            validate:"required"`
	Name *template.Template `yaml:"name,omitempty"`

	Subject *template.Template            `yaml:"subject,omitempty"`
	Payload *template.Template            `yaml:"payload,omitempty"`
	Headers map[string]*template.Template `yaml:"headers,omitempty"`

	Username    *template.Template `yaml:"username,omitempty"`
	Password    *template.Template `yaml:"password,omitempty"`
	Token       *template.Template `yaml:"token,omitempty"`
	Credentials *template.Template `yaml:"credentials,omitempty"`
	NKeySeed    *template.Template `yaml:"nkeySeed,omitempty"`
}

// setDefaults fills in the default subject, payload, and headers when unset.
func (c *NatsCommon) setDefaults() {
	if c.Subject == nil {
		t, _ := template.New(defaultNatsSubject)
		c.Subject = t
	}

	if c.Payload == nil {
		t, _ := template.New(defaultNatsPayload)
		c.Payload = t
	}

	if len(c.Headers) == 0 {
		c.Headers = map[string]*template.Template{}

		idempotency, _ := template.New(defaultNatsIdempotencyKey)
		c.Headers["X-Idempotency-Key"] = idempotency

		traceparent, _ := template.New(defaultNatsTraceparent)
		c.Headers["traceparent"] = traceparent
	}
}

// natsAuthValidator validates that at most one auth method is configured on the
// shared NATS connection fields.
func natsAuthValidator(sl validator.StructLevel) {
	c, ok := sl.Current().Interface().(NatsCommon)
	if !ok {
		return
	}

	groups := 0
	if c.Credentials != nil && c.Credentials.Source() != "" {
		groups++
	}
	if c.NKeySeed != nil && c.NKeySeed.Source() != "" {
		groups++
	}
	if (c.Username != nil && c.Username.Source() != "") || (c.Password != nil && c.Password.Source() != "") {
		groups++
	}
	if c.Token != nil && c.Token.Source() != "" {
		groups++
	}

	if groups > 1 {
		sl.ReportError(c, "auth", "Auth", "natsauth", "")
	}
}

func init() {
	Validate.RegisterStructValidation(natsAuthValidator, NatsCommon{})
}

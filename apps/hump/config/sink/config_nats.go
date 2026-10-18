package sink

import (
	_ "embed"

	"hump/config/decoder"
	"hump/pkg/template"

	"github.com/go-playground/validator/v10"
)

var (
	//go:embed templ/nats_subject.tmpl
	defaultNatsSubject string
)

// NatsCommon is the connection, auth and message settings shared by the core
// NATS and JetStream sinks; at most one auth method may be set.
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

// setDefaults fills in the default subject, payload and headers.
func (c *NatsCommon) setDefaults() {
	if c.Subject == nil {
		t, _ := template.New(defaultNatsSubject)
		c.Subject = t
	}

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
	decoder.Validate.RegisterStructValidation(natsAuthValidator, NatsCommon{})
}

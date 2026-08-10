package config

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"hump/internal/types/errs"
	"hump/internal/types/sink"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

var (
	//go:embed templ/rabbitmq_exchange.tmpl
	defaultRabbitMQExchange string

	//go:embed templ/rabbitmq_routing_key.tmpl
	defaultRabbitMQRoutingKey string
)

// RabbitMQExchangeMode selects how an AMQP exchange is reconciled at first
// publish.
type RabbitMQExchangeMode string

const (
	// RabbitMQExchangeModePassive asserts the exchange already exists and never
	// declares it.
	RabbitMQExchangeModePassive RabbitMQExchangeMode = "passive"

	// RabbitMQExchangeModeCreateWhenMissing declares the exchange on first
	// publish if it is absent.
	RabbitMQExchangeModeCreateWhenMissing RabbitMQExchangeMode = "create-when-missing"
)

// RabbitMQDeliveryMode selects whether published messages are persistent or
// transient.
type RabbitMQDeliveryMode string

const (
	// RabbitMQDeliveryPersistent marks messages for spooling to disk so they
	// survive a broker restart.
	RabbitMQDeliveryPersistent RabbitMQDeliveryMode = "persistent"

	// RabbitMQDeliveryTransient keeps messages in memory only; they are lost on
	// broker restart.
	RabbitMQDeliveryTransient RabbitMQDeliveryMode = "transient"
)

// RabbitMQExchange holds the exchange declaration knobs shared by every
// publish on this sink.
type RabbitMQExchange struct {
	Name       *template.Template   `yaml:"name,omitempty"`
	Type       string               `yaml:"type,omitempty" validate:"omitempty,oneof=direct topic fanout headers"`
	Durable    *bool                `yaml:"durable,omitempty"`
	AutoDelete bool                 `yaml:"autoDelete,omitempty"`
	Mode       RabbitMQExchangeMode `yaml:"mode,omitempty"       validate:"omitempty,oneof=passive create-when-missing"`
	Arguments  map[string]any       `yaml:"arguments,omitempty"`
}

// RabbitMQProperties mirrors the AMQP basic.properties fields hump exposes for
// templated rendering. Every entry is optional and rendered per publish.
type RabbitMQProperties struct {
	MessageID       *template.Template `yaml:"messageId,omitempty"`
	CorrelationID   *template.Template `yaml:"correlationId,omitempty"`
	Type            *template.Template `yaml:"type,omitempty"`
	AppID           *template.Template `yaml:"appId,omitempty"`
	UserID          *template.Template `yaml:"userId,omitempty"`
	ContentType     *template.Template `yaml:"contentType,omitempty"`
	ContentEncoding *template.Template `yaml:"contentEncoding,omitempty"`
	Expiration      *template.Template `yaml:"expiration,omitempty"`
	Priority        *template.Template `yaml:"priority,omitempty"`
}

// RabbitMQSink is the config for a classic AMQP sink.
type RabbitMQSink struct {
	RabbitMQCommon `yaml:",inline"`

	Exchange   RabbitMQExchange   `yaml:"exchange,omitempty"`
	RoutingKey *template.Template `yaml:"routingKey,omitempty"`

	Mandatory    *bool                `yaml:"mandatory,omitempty"`
	DeliveryMode RabbitMQDeliveryMode `yaml:"deliveryMode,omitempty" validate:"omitempty,oneof=persistent transient"`

	Properties RabbitMQProperties `yaml:"properties,omitempty"`
}

func (*RabbitMQSink) Type() sink.SinkType { return sink.SinkTypeRabbitMQ }

func (s *RabbitMQSink) DisplayName() string {
	if s == nil || s.URL == nil || s.URL.Source() == "" {
		return string(sink.SinkTypeRabbitMQ)
	}

	return fmt.Sprintf("%s(%s)", string(sink.SinkTypeRabbitMQ), s.URL.Source())
}

// Metadata returns the connection host (rendered URL) and the rendered exchange
// name for the given context.
func (s *RabbitMQSink) Metadata(context sink.SinkContext) (string, string) {
	if s == nil {
		return "", ""
	}

	url, _ := s.URL.Render(context)
	target, _ := s.Exchange.Name.Render(context)

	return parseHost(url), target
}

func (s *RabbitMQSink) UnmarshalYAML(node ast.Node) error {
	type rabbitmq RabbitMQSink
	if err := decode(node, (*rabbitmq)(s)); err != nil {
		return err
	}

	s.RabbitMQCommon.setDefaults()
	s.setDefaults()

	return nil
}

func (s *RabbitMQSink) setDefaults() {
	if s.Exchange.Name == nil {
		t, _ := template.New(defaultRabbitMQExchange)
		s.Exchange.Name = t
	}
	if s.Exchange.Type == "" {
		s.Exchange.Type = "topic"
	}
	if s.Exchange.Durable == nil {
		v := true
		s.Exchange.Durable = &v
	}
	if s.Exchange.Mode == "" {
		s.Exchange.Mode = RabbitMQExchangeModeCreateWhenMissing
	}

	if s.RoutingKey == nil {
		t, _ := template.New(defaultRabbitMQRoutingKey)
		s.RoutingKey = t
	}

	if s.Mandatory == nil {
		v := true
		s.Mandatory = &v
	}

	if s.DeliveryMode == "" {
		s.DeliveryMode = RabbitMQDeliveryPersistent
	}

	if s.Properties.MessageID == nil {
		t, _ := template.New(defaultRabbitMQIdempotencyKey)
		s.Properties.MessageID = t
	}
	if s.Properties.AppID == nil {
		t, _ := template.New("hump")
		s.Properties.AppID = t
	}
	if s.Properties.ContentType == nil {
		t, _ := template.New("application/json")
		s.Properties.ContentType = t
	}
}

// CrossValidate returns an error wrapping [errs.ErrCrossValidate] on the
// first (URL, exchange-name) pair whose declared type, durability,
// auto-delete, or arguments differ between two RabbitMQ sinks.
//
// The receiver is not inspected — the method's result depends only on tables.
func (*RabbitMQSink) CrossValidate(tables []Tables) error {
	type rmqEntry struct {
		exchangeType string
		durable      bool
		autoDelete   bool
		arguments    string
	}
	seen := make(map[string]rmqEntry)

	for _, table := range tables {
		for _, sc := range table.Sinks {
			rmqSink, ok := sc.Sink.(*RabbitMQSink)
			if !ok || rmqSink == nil {
				continue
			}

			url := ""
			if rmqSink.URL != nil {
				url = rmqSink.URL.Source()
			}
			exchangeName := ""
			if rmqSink.Exchange.Name != nil {
				exchangeName = rmqSink.Exchange.Name.Source()
			}
			key := fmt.Sprintf("%s\x00%s", url, exchangeName)

			argsJSON := "null"
			if len(rmqSink.Exchange.Arguments) > 0 {
				// encoding/json sorts map keys, so the byte string is canonical
				// across runs and deterministic across processes.
				b, err := json.Marshal(rmqSink.Exchange.Arguments)
				if err != nil {
					return fmt.Errorf("%w: exchange %q: encode arguments: %w", errs.ErrCrossValidate, exchangeName, err)
				}
				argsJSON = string(b)
			}

			durable := false
			if rmqSink.Exchange.Durable != nil {
				durable = *rmqSink.Exchange.Durable
			}
			e := rmqEntry{
				exchangeType: rmqSink.Exchange.Type,
				durable:      durable,
				autoDelete:   rmqSink.Exchange.AutoDelete,
				arguments:    argsJSON,
			}

			prev, exists := seen[key]
			if !exists {
				seen[key] = e
				continue
			}

			if prev != e {
				return fmt.Errorf("%w: exchange %q: divergent type/durable/autoDelete/arguments", errs.ErrCrossValidate, exchangeName)
			}
		}
	}

	return nil
}

package rabbitmq

import (
	"fmt"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"

	"hump/internal/domain"
	sink "hump/internal/port/sink"
	"hump/pkg/template"
)

// tableHeaders flattens an AMQP header table into a plain string map for the
// operations stream. It returns nil when there are no headers.
func tableHeaders(headers amqp.Table) domain.Headers {
	if len(headers) == 0 {
		return nil
	}

	out := make(domain.Headers, len(headers))
	for name, v := range headers {
		out[name] = fmt.Sprint(v)
	}
	return out
}

// rabbitMessage bundles the rendered exchange/routing key with the
// [amqp.Publishing] ready for [amqp.Channel.PublishWithContext].
type rabbitMessage struct {
	Exchange   string
	RoutingKey string
	Mandatory  bool
	Publishing amqp.Publishing
}

// buildMessage renders every templated field of cfg against operationContext
// and returns the assembled [rabbitMessage]. Empty-rendered headers are
// dropped (matches the Kafka/NATS sinks). DeliveryMode defaults to persistent;
// Priority is parsed from the rendered string and range-checked.
func buildMessage(cfg sink.RabbitMQSink, operationContext *sink.SinkContext) (*rabbitMessage, error) {
	exchange, err := cfg.Exchange.Name.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render exchange.name: %w", err)
	}

	routingKey, err := cfg.RoutingKey.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render routingKey: %w", err)
	}

	payload, err := cfg.Payload.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render payload: %w", err)
	}

	publishing := amqp.Publishing{
		Body:         []byte(payload),
		DeliveryMode: deliveryModeByte(cfg.DeliveryMode),
	}

	if err := renderProperties(&publishing, cfg.Properties, operationContext); err != nil {
		return nil, err
	}

	headers, err := renderHeaders(cfg.Headers, operationContext)
	if err != nil {
		return nil, err
	}
	publishing.Headers = headers

	mandatory := true
	if cfg.Mandatory != nil {
		mandatory = *cfg.Mandatory
	}

	return &rabbitMessage{
		Exchange:   exchange,
		RoutingKey: routingKey,
		Mandatory:  mandatory,
		Publishing: publishing,
	}, nil
}

// deliveryModeByte returns the AMQP wire-byte for the configured mode. An
// empty mode (caller bypassed setDefaults) maps to Persistent — same default
// as the YAML loader.
func deliveryModeByte(mode sink.RabbitMQDeliveryMode) uint8 {
	if mode == sink.RabbitMQDeliveryTransient {
		return amqp.Transient
	}
	return amqp.Persistent
}

// renderProperties renders each [sink.RabbitMQProperties] template and
// assigns the non-empty results to the matching [amqp.Publishing] field.
// Priority is parsed to uint8 and rejected if out of range.
func renderProperties(p *amqp.Publishing, props sink.RabbitMQProperties, opCtx *sink.SinkContext) error {
	var err error

	if p.MessageId, err = props.MessageID.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.messageId: %w", err)
	}
	if p.CorrelationId, err = props.CorrelationID.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.correlationId: %w", err)
	}
	if p.Type, err = props.Type.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.type: %w", err)
	}
	if p.AppId, err = props.AppID.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.appId: %w", err)
	}
	if p.UserId, err = props.UserID.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.userId: %w", err)
	}
	if p.ContentType, err = props.ContentType.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.contentType: %w", err)
	}
	if p.ContentEncoding, err = props.ContentEncoding.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.contentEncoding: %w", err)
	}
	if p.Expiration, err = props.Expiration.Render(opCtx); err != nil {
		return fmt.Errorf("render properties.expiration: %w", err)
	}

	priorityStr, err := props.Priority.Render(opCtx)
	if err != nil {
		return fmt.Errorf("render properties.priority: %w", err)
	}
	if priorityStr != "" {
		v, parseErr := strconv.ParseUint(priorityStr, 10, 8)
		if parseErr != nil {
			return fmt.Errorf("render properties.priority: invalid uint8 %q: %w", priorityStr, parseErr)
		}
		p.Priority = uint8(v)
	}

	return nil
}

// renderHeaders renders every templated header. Empty-rendered values are
// dropped, matching the Kafka/NATS conventions.
func renderHeaders(headers map[string]*template.Template, opCtx *sink.SinkContext) (amqp.Table, error) {
	if len(headers) == 0 {
		return nil, nil
	}

	out := make(amqp.Table, len(headers))
	for name, tmpl := range headers {
		v, err := tmpl.Render(opCtx)
		if err != nil {
			return nil, fmt.Errorf("render header %s: %w", name, err)
		}
		if len(v) == 0 {
			continue
		}
		out[name] = v
	}
	return out, nil
}

package rabbitmq_stream

import (
	"fmt"

	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	sink "hump/internal/port/sink"
)

// renderAll renders the templated stream name, payload, and headers from cfg
// against operationContext. Returns each rendered value separately so the
// caller can use them with the stream client's typed APIs.
func renderAll(cfg sinkcfg.RabbitMQStreamSink, operationContext *sink.Context) (streamName, payload string, headers domain.Headers, err error) {
	streamName, err = cfg.Stream.Name.Render(operationContext)
	if err != nil {
		err = fmt.Errorf("render stream.name: %w", err)
		return
	}

	payload, err = cfg.Payload.Render(operationContext)
	if err != nil {
		err = fmt.Errorf("render payload: %w", err)
		return
	}

	headers = make(domain.Headers, len(cfg.Headers))
	for name, tmpl := range cfg.Headers {
		v, hErr := tmpl.Render(operationContext)
		if hErr != nil {
			err = fmt.Errorf("render header %s: %w", name, hErr)
			return
		}
		headers[name] = v
	}

	return
}

// toAMQPProps copies the rendered headers into the AMQP 1.0
// application-properties map shape used by the stream client. Empty-rendered
// values are dropped.
func toAMQPProps(headers domain.Headers) map[string]any {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]any, len(headers))
	for name, v := range headers {
		if len(v) == 0 {
			continue
		}
		out[name] = v
	}
	return out
}

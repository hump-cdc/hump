package kafka

import (
	"fmt"
	"hump/internal/domain"

	"github.com/twmb/franz-go/pkg/kgo"
	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
)

// buildMessage renders the configured topic, partition key, payload, and
// headers against operationContext and returns the resulting [kgo.Record].
func buildMessage(config sinkcfg.KafkaSink, operationContext *sink.SinkContext) (*kgo.Record, error) {
	topic, err := config.Topic.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render topic: %w", err)
	}

	partitionKey, err := config.PartitionKey.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render partitionKey: %w", err)
	}

	payload, err := config.Payload.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render payload: %w", err)
	}

	record := &kgo.Record{
		Topic:   topic,
		Key:     []byte(partitionKey),
		Value:   []byte(payload),
		Headers: make([]kgo.RecordHeader, 0, len(config.Headers)),
	}

	for name, tmpl := range config.Headers {
		v, err := tmpl.Render(operationContext)
		if err != nil {
			return nil, fmt.Errorf("render header %s: %w", name, err)
		}
		if len(v) == 0 {
			continue
		}

		record.Headers = append(record.Headers, kgo.RecordHeader{Key: name, Value: []byte(v)})
	}

	return record, nil
}

// recordHeaders converts Kafka record headers into a plain string map for the
// operations stream. It returns nil when there are no headers.
func recordHeaders(headers []kgo.RecordHeader) domain.Headers {
	if len(headers) == 0 {
		return nil
	}

	out := make(domain.Headers, len(headers))
	for _, h := range headers {
		out[h.Key] = string(h.Value)
	}
	return out
}

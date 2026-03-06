package kafka

import (
	"fmt"
	"hump/internal/types/config"
	"hump/internal/types/sink"

	"github.com/twmb/franz-go/pkg/kgo"
)

// buildMessage renders the configured topic, partition key, payload, and
// headers against operationContext and returns the resulting [kgo.Record].
func buildMessage(config config.KafkaSink, operationContext *sink.SinkContext) (*kgo.Record, error) {
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

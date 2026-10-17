package kafka

import (
	"fmt"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"

	"github.com/twmb/franz-go/pkg/kgo"
)

// buildMessage renders the configured topic, partition key, payload, and
// headers against operationContext and returns the resulting [kgo.Record].
func buildMessage(config sinkcfg.KafkaSink, operationContext *sink.Context) (*kgo.Record, error) {
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

// recordHeaders records the Kafka record headers on capture, prefixed so they
// do not collide with the sink's own metadata.
func recordHeaders(headers []kgo.RecordHeader, capture *sink.Capture) {
	for _, h := range headers {
		capture.Set("header."+h.Key, string(h.Value))
	}
}

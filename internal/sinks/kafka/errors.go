package kafka

import (
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kerr"

	"hump/internal/types/errs"
	"hump/pkg/template"
)

// wrapError returns a "kafka" wrapped error, joining [errs.ErrSinkPermanent]
// for non-retriable errors: template render failures (shared across sinks) and
// the Kafka-specific produce errors classified by isPermanentKafkaError.
func wrapError(err error) error {
	if errors.Is(err, template.ErrRender) || isPermanentKafkaError(err) {
		return fmt.Errorf("kafka: %w", errors.Join(err, errs.ErrSinkPermanent))
	}
	return fmt.Errorf("kafka: %w", err)
}

// isPermanentKafkaError reports whether a Kafka produce error should bypass
// retries. UnknownTopicOrPartition is treated as permanent because in this
// producer's context it indicates a missing topic (a misconfiguration), not
// the transient leader/metadata propagation that Kafka's spec marks retriable.
func isPermanentKafkaError(err error) bool {
	var ke *kerr.Error
	if !errors.As(err, &ke) {
		return false
	}
	return ke == kerr.UnknownTopicOrPartition
}

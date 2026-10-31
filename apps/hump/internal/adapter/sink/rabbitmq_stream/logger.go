package rabbitmq_stream

import (
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/logs"

	"hump/pkg/observability/log"
)

var logger = log.Scope("rabbitmq-stream")

// dependencyLogger prevents the stream client's free-form diagnostics from exposing broker URLs.
type dependencyLogger struct{}

// init installs the logger before any stream client can start background goroutines.
func init() {
	logs.SetLogger(dependencyLogger{})
}

// Info emits a controlled client event without formatting untrusted text or arguments.
func (dependencyLogger) Info(_ string, _ ...any) {
	logger.Info("RabbitMQ stream client event")
}

// Error emits a controlled client error without formatting untrusted text or arguments.
func (dependencyLogger) Error(_ string, _ ...any) {
	logger.Error("RabbitMQ stream client error")
}

// Debug emits a controlled client debug event without formatting untrusted text or arguments.
func (dependencyLogger) Debug(_ string, _ ...any) {
	logger.Debug("RabbitMQ stream client debug event")
}

// Warn emits a controlled client warning without formatting untrusted text or arguments.
func (dependencyLogger) Warn(_ string, _ ...any) {
	logger.Warn("RabbitMQ stream client warning")
}

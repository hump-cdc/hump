package config

import (
	"net/url"
	"strings"
)

// parseHost extracts the "host:port" endpoint from a rendered sink URL.
// Schemed URLs (nats://, amqp://, https://, rabbitmq-stream://) return
// url.URL.Host, which already drops scheme, credentials, and path. Bare broker
// lists without a scheme (Kafka, e.g. "localhost:9092,localhost:9093") are
// returned verbatim.
func parseHost(rendered string) string {
	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		return ""
	}

	u, err := url.Parse(rendered)
	if err != nil || u.Host == "" {
		return rendered // bare broker list (kafka) or unparseable: keep as-is
	}

	return u.Host
}

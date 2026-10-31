// Package diagnostic provides fail-closed endpoint labels and errors for system diagnostics.
package diagnostic

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// hiddenEndpoint replaces endpoint inputs that cannot be safely identified.
const hiddenEndpoint = "[redacted endpoint]"

// safeError exposes controlled text while preserving its cause for errors.Is and errors.As.
type safeError struct {
	message string
	cause   error
}

// Endpoint returns only an allowed URL's scheme and authority, never credentials or resource names.
func Endpoint(raw string) string {
	if strings.ContainsAny(raw, "{},\\") || strings.IndexFunc(raw, unicode.IsSpace) >= 0 {
		return hiddenEndpoint
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || u.Host == "" {
		return hiddenEndpoint
	}
	switch u.Scheme {
	case "http", "https", "nats", "tls", "ws", "wss", "amqp", "amqps", "rabbitmq-stream", "rabbitmq-stream+tls":
	default:
		return hiddenEndpoint
	}
	host := u.Hostname()
	if host == "" || strings.ContainsAny(host, "%@:/?#[]") && net.ParseIP(host) == nil {
		return hiddenEndpoint
	}
	if strings.ContainsAny(u.Host, "[]") && net.ParseIP(host) == nil {
		return hiddenEndpoint
	}
	for _, r := range host {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', strings.ContainsRune("-._:", r):
		default:
			return hiddenEndpoint
		}
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return hiddenEndpoint
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return hiddenEndpoint
	}
	return u.Scheme + "://" + u.Host
}

// Endpoints returns safe labels for a comma-separated NATS server list.
func Endpoints(raw string) string {
	parts := strings.Split(raw, ",")
	for i, part := range parts {
		parts[i] = Endpoint(strings.TrimSpace(part))
	}
	return strings.Join(parts, ",")
}

// Error hides all cause text, retaining the cause only for programmatic inspection.
func Error(message string, cause error) error {
	if cause == nil {
		return nil
	}
	return &safeError{message: message, cause: cause}
}

// Error returns only the caller-controlled diagnostic message.
func (e *safeError) Error() string { return e.message }

// Unwrap preserves the original cause without formatting it.
func (e *safeError) Unwrap() error { return e.cause }

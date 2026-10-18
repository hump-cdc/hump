package metadata

import "testing"

func TestParseHost(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"nats", "nats://localhost:4222", "localhost:4222"},
		{"amqp with credentials", "amqp://hump:hump@localhost:5672/", "localhost:5672"},
		{"rabbitmq stream with vhost", "rabbitmq-stream://hump:hump@localhost:5552/%2F", "localhost:5552"},
		{"https with path", "https://webhooks.host/url1", "webhooks.host"},
		{"kafka single broker no scheme", "localhost:9092", "localhost:9092"},
		{"kafka multi broker no scheme", "localhost:9092,localhost:9093", "localhost:9092,localhost:9093"},
		{"empty", "", ""},
		{"whitespace", "  nats://localhost:4222  ", "localhost:4222"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseHost(tt.in); got != tt.want {
				t.Errorf("ParseHost(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

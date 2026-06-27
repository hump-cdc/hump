package config

import (
	"testing"

	"hump/internal/template"
	"hump/internal/types/sink"
)

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
			if got := parseHost(tt.in); got != tt.want {
				t.Errorf("parseHost(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func mustTemplate(t *testing.T, source string) *template.Template {
	t.Helper()
	tmpl, err := template.New(source)
	if err != nil {
		t.Fatalf("template.New(%q): %v", source, err)
	}
	return tmpl
}

func TestSinkMetadata(t *testing.T) {
	context := sink.SinkContext{Schema: "public", Table: "users"}

	tests := []struct {
		name       string
		sink       sink.Sink
		wantHost   string
		wantTarget string
	}{
		{
			name: "kafka",
			sink: &KafkaSink{
				URL:   mustTemplate(t, "localhost:9092,localhost:9093"),
				Topic: mustTemplate(t, "{{ .Schema }}.{{ .Table }}"),
			},
			wantHost:   "localhost:9092,localhost:9093",
			wantTarget: "public.users",
		},
		{
			name: "http",
			sink: &HTTPSink{
				URL:    mustTemplate(t, "https://webhooks.host/url1"),
				Method: mustTemplate(t, "POST"),
			},
			wantHost:   "webhooks.host",
			wantTarget: "POST",
		},
		{
			name: "nats core",
			sink: &NatsCoreSink{
				NatsCommon: NatsCommon{
					URL:     mustTemplate(t, "nats://localhost:4222"),
					Subject: mustTemplate(t, "{{ .Schema }}.{{ .Table }}"),
				},
			},
			wantHost:   "localhost:4222",
			wantTarget: "public.users",
		},
		{
			name: "nats jetstream",
			sink: &NatsJetStreamSink{
				NatsCommon: NatsCommon{
					URL:     mustTemplate(t, "nats://localhost:4222"),
					Subject: mustTemplate(t, "{{ .Schema }}.{{ .Table }}"),
				},
			},
			wantHost:   "localhost:4222",
			wantTarget: "public.users",
		},
		{
			name: "rabbitmq classic",
			sink: &RabbitMQSink{
				RabbitMQCommon: RabbitMQCommon{
					URL: mustTemplate(t, "amqp://hump:hump@localhost:5672/"),
				},
				Exchange: RabbitMQExchange{Name: mustTemplate(t, "hump.{{ .Schema }}")},
			},
			wantHost:   "localhost:5672",
			wantTarget: "hump.public",
		},
		{
			name: "rabbitmq stream",
			sink: &RabbitMQStreamSink{
				RabbitMQCommon: RabbitMQCommon{
					URL: mustTemplate(t, "rabbitmq-stream://hump:hump@localhost:5552/%2F"),
				},
				Stream: RabbitMQStreamConfig{Name: mustTemplate(t, "hump.{{ .Schema }}.{{ .Table }}")},
			},
			wantHost:   "localhost:5552",
			wantTarget: "hump.public.users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, target := tt.sink.Metadata(context)
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if target != tt.wantTarget {
				t.Errorf("target = %q, want %q", target, tt.wantTarget)
			}
		})
	}
}

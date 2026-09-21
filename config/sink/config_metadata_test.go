package sink

import (
	"testing"

	sinkport "hump/internal/port/sink"
	"hump/pkg/template"
)

func mustTemplate(t *testing.T, source string) *template.Template {
	t.Helper()
	tmpl, err := template.New(source)
	if err != nil {
		t.Fatalf("template.New(%q): %v", source, err)
	}
	return tmpl
}

func TestSinkMetadata(t *testing.T) {
	context := sinkport.SinkContext{Schema: "public", Table: "users"}

	tests := []struct {
		name       string
		sink       sinkport.Definition
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
			wantTarget: "https://webhooks.host/url1",
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

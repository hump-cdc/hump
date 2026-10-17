package sink

import (
	"testing"

	sinkport "hump/internal/port/sink"
)

func TestSinkTypeAndDisplayName(t *testing.T) {
	url := mustTemplate(t, "endpoint")

	tests := []struct {
		name      string
		populated sinkport.Definition
		nilRecv   sinkport.Definition
		wantType  sinkport.Type
	}{
		{name: "http", populated: &HTTPSink{URL: url}, nilRecv: (*HTTPSink)(nil), wantType: sinkport.SinkTypeHTTP},
		{name: "kafka", populated: &KafkaSink{URL: url}, nilRecv: (*KafkaSink)(nil), wantType: sinkport.SinkTypeKafka},
		{
			name:      "nats-core",
			populated: &NatsCoreSink{NatsCommon: NatsCommon{URL: url}},
			nilRecv:   (*NatsCoreSink)(nil),
			wantType:  sinkport.SinkTypeNatsCore,
		},
		{
			name:      "nats-jetstream",
			populated: &NatsJetStreamSink{NatsCommon: NatsCommon{URL: url}},
			nilRecv:   (*NatsJetStreamSink)(nil),
			wantType:  sinkport.SinkTypeNatsJetStream,
		},
		{
			name:      "rabbitmq",
			populated: &RabbitMQSink{RabbitMQCommon: RabbitMQCommon{URL: url}},
			nilRecv:   (*RabbitMQSink)(nil),
			wantType:  sinkport.SinkTypeRabbitMQ,
		},
		{
			name:      "rabbitmq-stream",
			populated: &RabbitMQStreamSink{RabbitMQCommon: RabbitMQCommon{URL: url}},
			nilRecv:   (*RabbitMQStreamSink)(nil),
			wantType:  sinkport.SinkTypeRabbitMQStream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.populated.Type(); got != tt.wantType {
				t.Errorf("Type = %q, want %q", got, tt.wantType)
			}
			if got, want := tt.populated.DisplayName(), string(tt.wantType)+"(endpoint)"; got != want {
				t.Errorf("DisplayName = %q, want %q", got, want)
			}
			if got := tt.nilRecv.DisplayName(); got != string(tt.wantType) {
				t.Errorf("nil DisplayName = %q, want %q", got, tt.wantType)
			}
		})
	}
}

func TestSinkMetadataNilReceiver(t *testing.T) {
	sinks := []sinkport.Definition{
		(*HTTPSink)(nil),
		(*KafkaSink)(nil),
		(*NatsCoreSink)(nil),
		(*NatsJetStreamSink)(nil),
		(*RabbitMQSink)(nil),
		(*RabbitMQStreamSink)(nil),
	}

	for _, s := range sinks {
		t.Run(string(s.Type()), func(t *testing.T) {
			host, target := s.Metadata(sinkport.Context{})
			if host != "" || target != "" {
				t.Fatalf("Metadata = (%q, %q), want empty", host, target)
			}
		})
	}
}

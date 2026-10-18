package sink

import (
	errs "errors"
	"testing"
	"time"

	"hump/config/errors"
	sinkport "hump/internal/port/sink"
)

func TestStatelessSinksCrossValidate(t *testing.T) {
	groups := [][]sinkport.Definition{{&HTTPSink{}, &KafkaSink{}, &NatsCoreSink{}}}

	for _, cv := range []CrossValidator{&HTTPSink{}, &KafkaSink{}, &NatsCoreSink{}} {
		if err := cv.CrossValidate(groups); err != nil {
			t.Errorf("%T.CrossValidate = %v, want nil", cv, err)
		}
	}
}

func TestNatsJetStreamCrossValidate(t *testing.T) {
	js := func(url string, subjects ...string) *NatsJetStreamSink {
		s := &NatsJetStreamSink{Stream: NatsStream{Name: "S", Subjects: subjects}}
		if url != "" {
			s.URL = mustTemplate(t, url)
		}
		return s
	}

	tests := []struct {
		name    string
		groups  [][]sinkport.Definition
		wantErr bool
	}{
		{
			name:   "other sinks and nil skipped",
			groups: [][]sinkport.Definition{{&HTTPSink{}, (*NatsJetStreamSink)(nil)}},
		},
		{
			name:   "same subjects in any order agree",
			groups: [][]sinkport.Definition{{js("nats://x", "a", "b")}, {js("nats://x", "b", "a")}},
		},
		{
			name:   "same stream on different servers is independent",
			groups: [][]sinkport.Definition{{js("nats://x", "a")}, {js("", "b")}},
		},
		{
			name:    "divergent subjects",
			groups:  [][]sinkport.Definition{{js("nats://x", "a")}, {js("nats://x", "b")}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCrossValidate(t, (*NatsJetStreamSink)(nil).CrossValidate(tt.groups), tt.wantErr)
		})
	}
}

func TestRabbitMQCrossValidate(t *testing.T) {
	durable := true
	rmq := func(exchangeType string, args map[string]any) *RabbitMQSink {
		return &RabbitMQSink{
			RabbitMQCommon: RabbitMQCommon{URL: mustTemplate(t, "amqp://x")},
			Exchange: RabbitMQExchange{
				Name:      mustTemplate(t, "ex"),
				Type:      exchangeType,
				Durable:   &durable,
				Arguments: args,
			},
		}
	}

	tests := []struct {
		name    string
		groups  [][]sinkport.Definition
		wantErr bool
	}{
		{
			name:   "other sinks and nil skipped",
			groups: [][]sinkport.Definition{{&HTTPSink{}, (*RabbitMQSink)(nil)}},
		},
		{
			name:   "unset URL, name and durable",
			groups: [][]sinkport.Definition{{&RabbitMQSink{}}, {&RabbitMQSink{}}},
		},
		{
			name: "same exchange agrees",
			groups: [][]sinkport.Definition{
				{rmq("topic", map[string]any{"a": 1})},
				{rmq("topic", map[string]any{"a": 1})},
			},
		},
		{
			name:    "divergent type",
			groups:  [][]sinkport.Definition{{rmq("topic", nil)}, {rmq("fanout", nil)}},
			wantErr: true,
		},
		{
			name:    "unencodable arguments",
			groups:  [][]sinkport.Definition{{rmq("topic", map[string]any{"a": make(chan int)})}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCrossValidate(t, (*RabbitMQSink)(nil).CrossValidate(tt.groups), tt.wantErr)
		})
	}
}

func TestRabbitMQStreamCrossValidate(t *testing.T) {
	stream := func(maxAge time.Duration) *RabbitMQStreamSink {
		return &RabbitMQStreamSink{
			RabbitMQCommon: RabbitMQCommon{URL: mustTemplate(t, "rabbitmq-stream://x")},
			Stream:         RabbitMQStreamConfig{Name: mustTemplate(t, "s"), MaxAge: maxAge},
		}
	}

	tests := []struct {
		name    string
		groups  [][]sinkport.Definition
		wantErr bool
	}{
		{
			name:   "other sinks and nil skipped",
			groups: [][]sinkport.Definition{{&HTTPSink{}, (*RabbitMQStreamSink)(nil)}},
		},
		{
			name:   "unset URL and name",
			groups: [][]sinkport.Definition{{&RabbitMQStreamSink{}}, {&RabbitMQStreamSink{}}},
		},
		{
			name:   "same stream agrees",
			groups: [][]sinkport.Definition{{stream(time.Hour)}, {stream(time.Hour)}},
		},
		{
			name:    "divergent maxAge",
			groups:  [][]sinkport.Definition{{stream(time.Hour)}, {stream(time.Minute)}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCrossValidate(t, (*RabbitMQStreamSink)(nil).CrossValidate(tt.groups), tt.wantErr)
		})
	}
}

// assertCrossValidate checks err is nil, or wraps ErrCrossValidate when wantErr.
func assertCrossValidate(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if !wantErr {
		if err != nil {
			t.Fatalf("CrossValidate = %v, want nil", err)
		}
		return
	}
	if !errs.Is(err, errors.ErrCrossValidate) {
		t.Fatalf("CrossValidate = %v, want ErrCrossValidate", err)
	}
}

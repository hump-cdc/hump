package sink

import (
	"context"
	"errors"
	"testing"

	sinkcfg "hump/config/sink"
	p "hump/internal/port/sink"
)

func TestProcessDispatchesByDefinitionType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	definitions := []p.Definition{
		&sinkcfg.KafkaSink{},
		&sinkcfg.NatsCoreSink{},
		&sinkcfg.NatsJetStreamSink{},
		&sinkcfg.HTTPSink{},
		&sinkcfg.RabbitMQSink{},
		&sinkcfg.RabbitMQStreamSink{},
	}
	for _, def := range definitions {
		t.Run(string(def.Type()), func(t *testing.T) {
			if err := New().Process(ctx, def, &p.Context{}, nil); !errors.Is(err, context.Canceled) {
				t.Errorf("Process(%T) error = %v, want the sink's context.Canceled", def, err)
			}
		})
	}
}

func TestProcessRejectsUnknownDefinition(t *testing.T) {
	err := New().Process(context.Background(), unknownDefinition{}, &p.Context{}, nil)
	if !errors.Is(err, p.ErrUnknownSinkType) {
		t.Fatalf("Process() error = %v, want ErrUnknownSinkType", err)
	}
}

// unknownDefinition is a sink definition no adapter handles.
type unknownDefinition struct{}

// Type returns a sink type no adapter is registered for.
func (unknownDefinition) Type() p.Type { return "unknown" }

// DisplayName returns the sink type.
func (unknownDefinition) DisplayName() string { return "unknown" }

// Metadata returns no labels.
func (unknownDefinition) Metadata(p.Context) (string, string) { return "", "" }

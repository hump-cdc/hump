package sink

import (
	"context"
)

type Definition interface {
	Type() SinkType
	DisplayName() string
	Metadata(context SinkContext) (host string, target string)
}

type Sink interface {
	// Process delivers operationContext to the sink described by config. A
	// non-nil capture is filled with what actually went on the wire.
	Process(ctx context.Context, config Definition, operationContext *SinkContext, capture *Capture) error
}

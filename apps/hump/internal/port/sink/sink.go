// Package sink is the delivery end of the pipeline: the sink adapter, the
// definitions it is configured with, and the row change it delivers.
package sink

import (
	"context"
)

// Definition is one configured sink, implemented by the YAML config structs.
// Metadata renders the metric host and target labels and may be called with
// only Schema and Table set.
type Definition interface {
	Type() Type
	DisplayName() string
	Metadata(context Context) (host string, target string)
}

// Sink delivers row changes to whichever sink a Definition describes; one
// instance serves every configured sink.
type Sink interface {
	// Process delivers operationContext to the sink described by config. A
	// non-nil capture is filled with what actually went on the wire.
	Process(ctx context.Context, config Definition, operationContext *Context, capture *Capture) error
}

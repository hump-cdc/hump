package sink

import (
	"context"
	"hump/internal/types/stream"
)

type Definition interface {
	Type() SinkType
	DisplayName() string
	Metadata(context SinkContext) (host string, target string)
}

type Sink interface {
	Process(ctx context.Context, config Definition, operationContext *SinkContext, streamEvent *stream.Event) error
}

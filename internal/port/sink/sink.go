package sink

import (
	"context"
	"hump/internal/types/sink"
	"hump/internal/types/stream"
)

type Sink interface {
	Process(ctx context.Context, config any, operationContext *sink.SinkContext, streamEvent *stream.Event) error
}

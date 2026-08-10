package routes

import (
	"context"
	"net/http"

	"hump/internal/types/stream"
	"hump/internal/web/streamer"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
)

// RegisterOperations registers the SSE endpoint that streams change-data-capture
// operations to connected clients. Each connection gets its own subscription on
// the streamer, which is removed when the client disconnects.
func RegisterOperations(api huma.API, s *streamer.FanOutStreamer[stream.Message]) {
	sse.Register(api, huma.Operation{
		OperationID: "operations.stream",
		Method:      http.MethodGet,
		Path:        "/api/operations",
		Summary:     "Stream change-data-capture operations",
		Tags:        []string{"Operations"},
	}, map[string]any{
		"operation": stream.Event{},
		"retry":     stream.RetryEvent{},
	}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if s == nil {
			return
		}

		id, ch := s.Subscribe()
		defer s.Unsubscribe(id)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				if err := send(sse.Message{Data: event}); err != nil {
					return
				}
			}
		}
	})
}

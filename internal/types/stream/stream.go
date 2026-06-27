// Package stream defines the operation event streamed to web clients over SSE
// and the publisher interface used to emit those events. It is a leaf package so
// the sink handlers can publish without importing internal/web.
package stream

import (
	"time"

	"hump/internal/types/wal"
)

// Event is a single change-data-capture operation delivered to a sink, shaped
// for the live operations stream consumed by the web UI. It is emitted once per
// operation, after the first publish attempt, carrying that attempt's outcome.
type Event struct {
	ID        string           `json:"id"`
	Timestamp time.Time        `json:"timestamp"`
	Schema    string           `json:"schema"`
	Table     string           `json:"table"`
	Operation wal.WalOperation `json:"operation"`
	SinkType  string           `json:"sinkType"`
	Host      string           `json:"host"`
	Target    string           `json:"target"`

	Headers map[string]string `json:"headers,omitempty"`
	Payload string            `json:"payload,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	DurationMs int32  `json:"durationMs"`
}

// RetryEvent is a single retry attempt for an operation, linked back to the
// originating Event by OperationID.
type RetryEvent struct {
	OperationID string    `json:"operationId"`
	Attempt     int       `json:"attempt"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	DurationMs  int32     `json:"durationMs"`
}

// Message is an event delivered on the operations stream: either an Event or a
// RetryEvent. The unexported marker method seals the set of implementations so
// only these two types can flow through the stream.
type Message interface {
	isStreamMessage()
}

func (Event) isStreamMessage()      {}
func (RetryEvent) isStreamMessage() {}

// Publisher fans an event (an Event or a RetryEvent) out to every connected SSE
// client. HasSubscribers lets producers skip building events when nobody is
// listening.
type Publisher interface {
	Publish(event Message)
	HasSubscribers() bool
}

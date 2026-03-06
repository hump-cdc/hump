package wal

import (
	"maps"
	"sync"
)

// messageContext is a thread-safe store for messages emitted via
// pg_logical_emit_message() and decoded from the WAL stream. It buffers
// name/value pairs that arrive between transaction boundaries so they can be
// attached to subsequent change events.
type messageContext struct {
	mu       sync.Mutex
	messages map[string]string
}

// Append stores a message under the given name, overwriting any prior value.
func (c *messageContext) Append(name, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages[name] = value
}

// Flush returns the buffered messages and resets the internal store.
func (c *messageContext) Flush() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := c.messages
	c.messages = map[string]string{}

	return messages
}

// Dump returns a copy of the buffered messages without clearing the store.
func (c *messageContext) Dump() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make(map[string]string, len(c.messages))
	maps.Copy(messages, c.messages)

	return messages
}

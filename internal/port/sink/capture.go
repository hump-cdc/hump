package sink

// Capture collects what an adapter actually sent and what came back, for the
// operations log. It is nil unless a client is watching, and an adapter must
// then skip the work of filling it. An adapter fills it after publishing, on
// both the success and the error path, so a failed delivery is still explained
// by its payload.
//
// The buffers belong to the caller and are reused, so an adapter appends into
// Payload[:0] and writes into Metadata rather than replacing either.
type Capture struct {
	Target   string            // Target is the destination the sink rendered for this row: topic, subject, exchange, stream or URL.
	Payload  []byte            // Payload is the message body as it went on the wire.
	Metadata map[string]string // Metadata is whatever else the sink knows about the call: headers, HTTP method and status, acknowledgements.
}

// Set records a metadata entry, allocating the map on first use.
func (c *Capture) Set(key, value string) {
	if c.Metadata == nil {
		c.Metadata = make(map[string]string, 8)
	}

	c.Metadata[key] = value
}

// SetPayload copies body into the reused payload buffer.
func (c *Capture) SetPayload(body []byte) {
	c.Payload = append(c.Payload[:0], body...)
}

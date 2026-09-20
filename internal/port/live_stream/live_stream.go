// Package livestream defines the live operations feed the web UI subscribes to
// over SSE: one finished delivery reduced to where it went, and the batch of
// them one window closes with.
package livestream

// Operation is one finished delivery, reduced to the fields the live feed
// shows. It carries no payload, no timing and no outcome: the feed says what
// moved, and the operations log answers everything else about it.
type Operation struct {
	Schema string
	Table  string
	Sink   string
	Host   string
	Target string
}

// Batch is every operation recorded since the previous batch, delivered as a
// single SSE event so a busy pipeline cannot outpace the client.
type Batch struct {
	Operations []Operation
}

// Streamer subscribes a web client to the batched operations feed. A
// subscriber that cannot keep up loses batches rather than slowing the
// pipeline down.
type Streamer interface {
	// Subscribe registers a client and returns its id together with the
	// channel its batches arrive on. A closed streamer answers a closed
	// channel, so a late caller exits at once.
	Subscribe() (uint64, <-chan Batch)
	// Unsubscribe removes the subscriber with id and closes its channel. It is
	// safe to call more than once.
	Unsubscribe(id uint64)
}

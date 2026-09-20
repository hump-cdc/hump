// Package operationscollector defines the in-memory log of finished sink
// deliveries shown on the web UI. It keeps the newest records only, in a ring
// of fixed capacity, and only while a client is watching.
package operationscollector

import (
	"time"

	"hump/internal/domain"
)

// Attempt is one delivery attempt of an operation. It uses the same status
// vocabulary as the sink metrics: success for the attempt that delivered,
// retry for a failed one the loop repeated, failure for the last failed one.
type Attempt struct {
	StartedAt time.Time
	Duration  time.Duration
	Status    domain.DeliveryStatus
	Error     string // Error is empty when the attempt succeeded.
}

// Record is one delivery of one row to one sink, with every attempt it took.
// A retry is an element of Attempts rather than a record of its own.
type Record struct {
	Seq         uint64        // Seq orders records across the whole process; the collector assigns it.
	StartedAt   time.Time     // StartedAt is when the first attempt began.
	Duration    time.Duration // Duration covers every attempt and every backoff pause.
	Timestamp   time.Time     // Timestamp is the WAL commit time of the row.
	Database    domain.DbName
	Table       domain.TableIdentifier
	Operation   domain.OperationType
	SinkType    string
	Host        string
	Target      string                // Target is what the sink rendered for this row, which a templated topic or URL makes differ from the configured one.
	Status      domain.DeliveryStatus // Status is how the delivery ended once the retry loop was done: success or failure. Retry lives on the attempts.
	Error       string                // Error is the terminal error of the retry loop; it is empty when the delivery succeeded.
	Retries     int                   // Retries is len(Attempts)-1, so a list can say "failed after 4 retries" without walking them.
	Payload     []byte                // Payload is the body as it went on the wire, truncated to the collector's limit.
	PayloadSize int                   // PayloadSize is the untruncated body size in bytes.
	Metadata    map[string]string     // Metadata is what the sink reported about the call: headers, HTTP method and status, acknowledgements.
	Attempts    []Attempt
}

// Query selects a window of the log. Every filter is optional: an empty one
// matches every record, so the zero Query is the first page of the whole ring.
// A list filter matches a record carrying any one of its values.
type Query struct {
	SinkTypes  []string // SinkTypes must contain the record's sink type exactly.
	Tables     []string // Tables are schema-qualified names, "schema.name".
	Operations []domain.OperationType
	Statuses   []domain.DeliveryStatus
	Text       string // Text is an ASCII case-insensitive substring of the payload, target, errors or metadata; the payload is only searched as far as the collector kept it.
	Start, End int    // Start and End are indices into the matching records, counted from the newest, cutting out the half-open window [Start, End).
}

// Page is one window of the log, newest first.
type Page struct {
	Records []Record
	Start   int // Start and End are the offsets the window was cut at, clamped to Total.
	End     int
	Total   int    // Total is how many records matched the query, which is everything the ring holds when the query has no filter.
	Written uint64 // Written counts every record stored since the collector last activated, so a reader can tell it fell behind.
}

// Collector keeps the newest finished deliveries in a ring buffer while at
// least one client is connected, and for an idle period after the last leaves.
type Collector interface {
	// Connect registers a client, allocating the ring if the collector was idle.
	Connect()
	// Disconnect unregisters a client previously registered with Connect.
	Disconnect()
	// Connected reports whether a client is attached. Callers check it before
	// doing any work for a record, so an idle collector costs one atomic load.
	// It is a hint: the last client may leave before the Record that follows.
	Connected() bool
	// Record stores rec. It takes over rec's Payload, Metadata and Attempts
	// buffers and hands back reusable ones in their place, so the caller must
	// not retain them across the call. It does nothing while idle.
	Record(rec *Record)
	// Dump returns the records in [start, end) counted from the newest, with
	// the totals needed to page through the rest. Both offsets are clamped to
	// Total, so an idle collector answers an empty page rather than an error.
	// Every buffer is copied, since the ring overwrites its slots afterwards.
	Dump(start, end int) Page
	// Search is Dump over the records matching q, which are counted into the
	// page's Total before its window is cut out of them.
	Search(q Query) Page
}

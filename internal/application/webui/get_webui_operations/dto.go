package getwebuioperations

import "time"

// OperationsRequest is the filter one page of the log is cut with. Every field
// but the window is optional, and an omitted one matches every record.
type OperationsRequest struct {
	Body struct {
		From       int      `json:"from"                 required:"true" minimum:"0" doc:"Index of the first record to return, counted from the newest match, so 0 is the most recent delivery"`
		To         int      `json:"to"                   required:"true" minimum:"0" doc:"Index one past the last record to return, so from 0 to 50 asks for the 50 newest matches"`
		Sink       string   `json:"sink,omitempty"       doc:"Sink type to filter by"`
		Table      string   `json:"table,omitempty"      doc:"Schema-qualified table to filter by"`
		Operations []string `json:"operations,omitempty" enum:"insert,update,delete,truncate" doc:"Row operations to filter by; an empty list matches all of them"`
		Status     string   `json:"status,omitempty"     enum:"success,failure" doc:"Terminal outcome to filter by; retry is an attempt status and never a record's"`
		Text       string   `json:"text,omitempty"       doc:"Case-insensitive substring of the payload, target, error or metadata"`
	}
}

// OperationAttempt is one delivery attempt of an operation.
type OperationAttempt struct {
	Attempt   int       `json:"attempt"         doc:"Position of the attempt, counted from 1"`
	StartedAt time.Time `json:"startedAt"       doc:"When the attempt began"`
	Duration  int64     `json:"duration"        doc:"How long the attempt took in milliseconds, the timeout wait included"`
	Status    string    `json:"status"          doc:"How the attempt ended: success, retry or failure"`
	Error     string    `json:"error,omitempty" doc:"Error the attempt failed with; absent when it succeeded"`
}

// OperationItem is one delivery of one row to one sink, with every attempt it
// took. A retry is an element of attempts rather than an item of its own.
type OperationItem struct {
	Seq         uint64             `json:"seq"             doc:"Number ordering the record across the whole process"`
	StartedAt   time.Time          `json:"startedAt"       doc:"When the first attempt began"`
	Duration    int64              `json:"duration"        doc:"How long the delivery took in milliseconds, every attempt and backoff pause included"`
	Timestamp   time.Time          `json:"timestamp"       doc:"WAL commit time of the row"`
	Database    string             `json:"database"        doc:"Source database"`
	Table       string             `json:"table"           doc:"Schema-qualified source table"`
	Operation   string             `json:"operation"       doc:"Row operation: insert, update, delete or truncate"`
	Sink        string             `json:"sink"            doc:"Sink type"`
	Host        string             `json:"host"            doc:"Connection host the sink URL renders to"`
	Target      string             `json:"target"          doc:"Target the sink rendered for this row: topic, subject, exchange, stream or URL"`
	Status      string             `json:"status"          doc:"How the delivery ended: success or failure"`
	Error       string             `json:"error,omitempty" doc:"Error the delivery ended with; absent when it succeeded"`
	Retries     int                `json:"retries"         doc:"Attempts beyond the first"`
	Payload     string             `json:"payload"         doc:"Body as it went on the wire, truncated to the collector's limit"`
	PayloadSize int                `json:"payloadSize"     doc:"Untruncated body size in bytes"`
	Metadata    map[string]string  `json:"metadata"        doc:"What the sink reported about the call: headers under a header. prefix, HTTP method and status, acknowledgements"`
	Attempts    []OperationAttempt `json:"attempts"        doc:"Every attempt the delivery took, oldest first"`
}

type OperationsResponse struct {
	Body struct {
		Total   int             `json:"total"   doc:"Records matching the filter, of which the window is one page"`
		From    int             `json:"from"    doc:"Index the window starts at, clamped to total"`
		To      int             `json:"to"      doc:"Index the window ends before, clamped to total"`
		Written uint64          `json:"written" doc:"Records stored since the log started, so a reader can tell the ring overwrote what it had not read"`
		Items   []OperationItem `json:"items"   doc:"The window, strictly newest first: index 0 is the most recent delivery and the last item the oldest"`
	}
}

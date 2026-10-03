package streamsnapshot

import "time"

// SnapshotRunSink is one sink a table is delivered to.
type SnapshotRunSink struct {
	Type   string `json:"type"   doc:"Sink type the rows are delivered to"`
	Host   string `json:"host"   doc:"Connection host the sink URL renders to"`
	Target string `json:"target" doc:"Target the sink renders for the table: topic, subject, exchange, stream or URL"`
	Count  uint64 `json:"count"  doc:"Rows delivered to the sink so far"`
}

// SnapshotProcessingTable is the progress of one table of a running snapshot.
type SnapshotProcessingTable struct {
	Schema        string            `json:"schema"          doc:"Schema of the table"`
	Name          string            `json:"name"            doc:"Name of the table"`
	Where         string            `json:"where,omitempty" doc:"Boolean expression the rows are filtered with, as it would follow WHERE; empty reads every row"`
	Status        string            `json:"status"          enum:"wait,processing,done" doc:"Phase of the table: waiting for its turn, being read, or read"`
	Rows          uint64            `json:"rows"            doc:"Rows read so far"`
	EstimatedRows uint64            `json:"estimatedRows"   doc:"Rows the snapshot expects to read"`
	Sinks         []SnapshotRunSink `json:"sinks"           doc:"Sinks the table is delivered to"`
}

// SnapshotDoneTable is the outcome of one table of a finished snapshot.
type SnapshotDoneTable struct {
	Schema string            `json:"schema"          doc:"Schema of the table"`
	Name   string            `json:"name"            doc:"Name of the table"`
	Status string            `json:"status"          enum:"success,error" doc:"Whether the table was read completely"`
	Error  string            `json:"error,omitempty" doc:"Why the table was not read completely"`
	Rows   uint64            `json:"rows"            doc:"Rows read"`
	Sinks  []SnapshotRunSink `json:"sinks"           doc:"Sinks the table was delivered to"`
}

// SnapshotIdleEvent is the state of a server that has not run a snapshot yet.
type SnapshotIdleEvent struct {
	State string `json:"state" enum:"idle" doc:"Snapshot state"`
}

// SnapshotProcessingEvent is the state of a snapshot in progress, sent on every change.
type SnapshotProcessingEvent struct {
	State     string                    `json:"state"     enum:"processing" doc:"Snapshot state"`
	StartedAt time.Time                 `json:"startedAt" doc:"When the snapshot started, UTC"`
	Tables    []SnapshotProcessingTable `json:"tables"    doc:"Tables of the snapshot, in the order they were requested"`
}

// SnapshotDoneEvent is the state of a finished snapshot, successful or not.
type SnapshotDoneEvent struct {
	State      string              `json:"state"      enum:"done" doc:"Snapshot state"`
	StartedAt  time.Time           `json:"startedAt"  doc:"When the snapshot started, UTC"`
	FinishedAt time.Time           `json:"finishedAt" doc:"When the snapshot finished, UTC"`
	Tables     []SnapshotDoneTable `json:"tables"     doc:"Tables of the snapshot, in the order they were requested"`
}

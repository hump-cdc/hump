package getsnapshottables

// SnapshotColumn is one column of a table.
type SnapshotColumn struct {
	Name       string `json:"name"       doc:"Column name"`
	Type       string `json:"type"       doc:"Column type as PostgreSQL formats it, e.g. timestamp with time zone"`
	Nullable   bool   `json:"nullable"   doc:"Whether the column accepts NULL"`
	PrimaryKey bool   `json:"primaryKey" doc:"Whether the column is part of the primary key"`
}

// SnapshotSink is one sink and the target it renders for a table.
type SnapshotSink struct {
	Type   string `json:"type"   doc:"Sink type, e.g. kafka or nats"`
	Host   string `json:"host"   doc:"Connection host the sink URL renders to"`
	Target string `json:"target" doc:"Target rendered for the table: topic, subject, exchange, stream or URL"`
}

// SnapshotTable is one configured table and the filter suggested for its snapshot.
type SnapshotTable struct {
	Schema  string           `json:"schema"  doc:"Schema of the table"`
	Table   string           `json:"table"   doc:"Name of the table"`
	Type    string           `json:"type"    doc:"How the table stores its rows" enum:"regular,partitioned,hypertable"`
	Columns []SnapshotColumn `json:"columns" doc:"Columns of the table, in table order"`
	Where   string           `json:"where"   doc:"Suggested filter keeping recent rows, as it would follow WHERE; empty suggests none"`
	Sinks   []SnapshotSink   `json:"sinks"   doc:"Sinks the table is delivered to, one entry per distinct sink and target"`
}

// SnapshotTablesResponse lists the configured tables found in the database.
type SnapshotTablesResponse struct {
	Body struct {
		Tables []SnapshotTable `json:"tables" doc:"Configured tables found in the database, in configuration order"`
	}
}

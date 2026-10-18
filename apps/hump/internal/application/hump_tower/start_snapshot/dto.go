package startsnapshot

// SnapshotStartTable is one table to snapshot and the filter its rows are read with.
type SnapshotStartTable struct {
	Schema string `json:"schema,omitempty" doc:"Schema of the table; empty resolves the table through search_path"`
	Table  string `json:"table"            required:"true" minLength:"1" doc:"Name of the table"`
	Where  string `json:"where,omitempty"  doc:"Boolean expression the rows are filtered with, as it would follow WHERE; empty reads every row"`
}

// SnapshotStartRequest lists the tables to snapshot.
type SnapshotStartRequest struct {
	Body struct {
		Tables []SnapshotStartTable `json:"tables" required:"true" minItems:"1" doc:"Tables to snapshot"`
	}
}

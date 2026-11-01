package startsnapshot

// SnapshotStartTable is one table to snapshot and the filter its rows are read with.
type SnapshotStartTable struct {
	Schema string `json:"schema,omitempty" doc:"Exact configured schema; empty requires an unambiguous table name with an explicit schema in configuration"`
	Table  string `json:"table"            required:"true" minLength:"1" doc:"Name of the table"`
	Where  string `json:"where,omitempty"  doc:"Boolean expression the rows are filtered with, as it would follow WHERE; empty reads every row"`
}

// SnapshotStartRequest lists the tables to snapshot.
type SnapshotStartRequest struct {
	Body struct {
		Tables []SnapshotStartTable `json:"tables" required:"true" minItems:"1" doc:"Tables to snapshot"`
	}
}

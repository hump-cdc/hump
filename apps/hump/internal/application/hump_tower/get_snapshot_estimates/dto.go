package getsnapshotestimates

// EstimateTable is one table to estimate and the filter its rows are counted with.
type EstimateTable struct {
	Schema string `json:"schema,omitempty" doc:"Schema of the table; empty resolves the table through search_path"`
	Table  string `json:"table"            required:"true" minLength:"1" doc:"Name of the table"`
	Where  string `json:"where,omitempty"  doc:"Boolean expression the rows are filtered with, as it would follow WHERE; empty counts every row"`
}

// EstimatesRequest lists the tables to estimate.
type EstimatesRequest struct {
	Body struct {
		Tables []EstimateTable `json:"tables" required:"true" doc:"Tables to estimate, each counted independently"`
	}
}

// EstimateEvent is one row count of one table: the planner's estimate first,
// then the exact count unless it timed out.
type EstimateEvent struct {
	Schema string `json:"schema" doc:"Schema of the table, as requested"`
	Table  string `json:"table"  doc:"Name of the table, as requested"`
	Count  uint64 `json:"count"  doc:"Number of rows matching the filter"`
	Rough  bool   `json:"rough"  doc:"Whether count is the planner's estimate rather than an exact count"`
}

// EstimateErrorEvent reports a table that could not be counted.
type EstimateErrorEvent struct {
	Schema string `json:"schema" doc:"Schema of the table, as requested"`
	Table  string `json:"table"  doc:"Name of the table, as requested"`
	Error  string `json:"error"  doc:"Why the table could not be counted: an invalid filter, a missing table or a failed query"`
}

// EstimatesDoneEvent closes the stream once every table has been counted.
type EstimatesDoneEvent struct{}

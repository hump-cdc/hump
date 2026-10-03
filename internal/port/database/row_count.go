package database

// RowCount is one estimate of how many rows of a table match a filter.
type RowCount struct {
	Num   uint64
	Rough bool  // Rough marks a planner estimate rather than an exact count.
	Error error // Error, when set, replaces the estimate: the count could not be taken.
}

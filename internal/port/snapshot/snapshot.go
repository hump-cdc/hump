// Package snapshot defines the snapshot run the web UI starts, stops and
// watches over SSE: one run at a time, re-reading the chosen tables and
// fanning their rows out to the sinks.
package snapshot

import (
	"errors"
	"time"

	"hump/internal/domain"
)

// Status is the phase the snapshot runner is in.
type Status string

const (
	// StatusIdle is a runner with no snapshot to report: none ran yet, or the
	// finished one was cleared by Stop.
	StatusIdle Status = "idle"
	// StatusProcessing is a snapshot in progress.
	StatusProcessing Status = "processing"
	// StatusDone is a finished snapshot, successful or not.
	StatusDone Status = "done"
)

// TableStatus is the phase one table of a run is in. A processing run uses
// wait, processing and done; a finished run reports success or error.
type TableStatus string

const (
	TableWait       TableStatus = "wait"
	TableProcessing TableStatus = "processing"
	TableDone       TableStatus = "done"
	TableSuccess    TableStatus = "success"
	TableError      TableStatus = "error"
)

var (
	// ErrAlreadyRunning is returned by Start while a snapshot is in progress.
	ErrAlreadyRunning = errors.New("snapshot is already running")
	// ErrNotRunning is returned by Stop when the runner is idle.
	ErrNotRunning = errors.New("snapshot is not running")
	// ErrInvalidWhere is returned by Start when a table's filter is rejected.
	ErrInvalidWhere = errors.New("invalid where expression")
	// ErrClosed is returned by Start once the runner has shut down.
	ErrClosed = errors.New("snapshot runner is closed")
)

// Sink is one sink a table is delivered to, with the rows it received so far.
type Sink struct {
	Type   string
	Host   string
	Target string
	Count  uint64
}

// Table is the progress of one table of a run.
type Table struct {
	Table         domain.TableIdentifier
	Where         string // Where is the filter the rows are read with; empty reads every row.
	Status        TableStatus
	Error         string // Error is set on a finished table that failed.
	CompletedRows uint64
	EstimatedRows uint64 // EstimatedRows is the row count the run expects to read; set while processing.
	Sinks         []Sink
}

// State is the whole state of the runner, sent in full on every change.
type State struct {
	Status     Status
	StartedAt  time.Time // StartedAt is zero while idle.
	FinishedAt time.Time // FinishedAt is set once done.
	Tables     []Table
}

// Request is one table to snapshot and the filter its rows are read with.
type Request struct {
	Table domain.TableIdentifier
	Where string
}

// Runner runs one snapshot at a time and streams its state to subscribers.
type Runner interface {
	// Subscribe registers a client and returns its id, the current state and
	// the channel later states arrive on. A subscriber that falls behind gets
	// the latest state rather than every one. A closed runner answers a closed
	// channel.
	Subscribe() (uint64, State, <-chan State)
	// Unsubscribe removes the subscriber with id and closes its channel. It is
	// safe to call more than once.
	Unsubscribe(id uint64)
	// Start begins a snapshot of tables, or returns ErrAlreadyRunning, ErrClosed
	// after shutdown, or ErrInvalidWhere when any filter is rejected, before
	// anything starts.
	Start(tables []Request) error
	// Stop ends the snapshot in progress, failing the tables it did not
	// finish, clears a finished one back to idle, or returns ErrNotRunning.
	Stop() error
}

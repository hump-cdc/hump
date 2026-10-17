package database

import "errors"

var (
	// ErrDbSetup wraps every failure to read the server's settings or to meet
	// the requirements for logical replication.
	ErrDbSetup = errors.New("database configuration error")
	// ErrWal wraps every failure of the replication stream, from starting it to
	// decoding a message off it.
	ErrWal = errors.New("WAL error")
	// ErrNoTablesConfigured indicates that the configuration declares no
	// tables to replicate.
	ErrNoTablesConfigured = errors.New("no tables configured for the service")
	// ErrTablesNotFound indicates that one or more configured tables do not
	// exist in the live database schema.
	ErrTablesNotFound = errors.New("requested tables are not in database schema")
	// ErrWalClosed indicates that the walsender connection was closed.
	ErrWalClosed = errors.New("wal sender connection closed")
	// ErrInvalidWhere indicates that a row filter is not a plain WHERE
	// expression hump accepts.
	ErrInvalidWhere = errors.New("invalid where expression")
	// ErrTableNotFound indicates that a requested table does not exist.
	ErrTableNotFound = errors.New("table not found")
	// ErrSnapshot wraps every failure to export, import or read a snapshot.
	ErrSnapshot = errors.New("snapshot error")
)

// Package schema defines the PostgreSQL table metadata types used across the
// source pipeline (table classification, partition lookups, sentinel errors).
package schema

import "errors"

var (
	// ErrNoTablesConfigured indicates that the configuration declares no
	// tables to replicate.
	ErrNoTablesConfigured = errors.New("no tables configured for the service")
	// ErrTablesNotFound indicates that one or more configured tables do not
	// exist in the live database schema.
	ErrTablesNotFound = errors.New("requested tables are not in database schema")
)

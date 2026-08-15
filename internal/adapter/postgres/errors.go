package postgres

import "errors"

// ErrTableIgnored indicates that a table has no configured sinks and that
// introspection will not resolve one, so changes to it are skipped.
var ErrTableIgnored = errors.New("table is ignored")

// errIntrospectionRequired indicates that the in-memory registry cannot answer
// a lookup and only a catalog refresh can.
var errIntrospectionRequired = errors.New("introspection required")

package postgres

import "errors"

// ErrTableIgnored indicates that a table has no configured sinks and that
// introspection will not resolve one, so changes to it are skipped.
var ErrTableIgnored = errors.New("table is ignored")

// errIntrospectionRequired indicates that the in-memory registry cannot answer
// a lookup and only a catalog refresh can.
var errIntrospectionRequired = errors.New("introspection required")

// errEmptyCopyData indicates the WAL sender sent a CopyData frame with an empty
// body, which carries no message type byte to dispatch on.
var errEmptyCopyData = errors.New("received empty CopyData frame")

// errEmptyWalRecord indicates an XLogData frame carried no logical decoding
// message, leaving nothing to parse.
var errEmptyWalRecord = errors.New("empty logical replication message")

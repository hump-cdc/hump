package shutdown

import "errors"

var (
	// ErrShutdownTimeout is reported for a hook that outlived its timeout.
	ErrShutdownTimeout = errors.New("service shutdown timeout")
)

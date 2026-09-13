package shutdown

import "errors"

var (
	ErrShutdownTimeout = errors.New("service shutdown timeout")
)

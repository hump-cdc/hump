package app

import "errors"

var (
	// ErrShutdown is returned by a Run function that finished its job and wants
	// the process to exit cleanly.
	ErrShutdown = errors.New("shutdown requested")
	// ErrDrainTimeout reports that Run functions were still busy when the drain
	// deadline passed.
	ErrDrainTimeout = errors.New("drain timeout: in-flight work did not finish")
)

package app

import "errors"

var (
	ErrShutdown = errors.New("shutdown requested")
)

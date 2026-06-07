package errs

import "errors"

var (
	ErrDbSetup                 = errors.New("database configuration error")
	ErrConfigurationParse      = errors.New("error parsing YAML configuration")
	ErrUnknownSinkType         = errors.New("unknown sink type")
	ErrSinkPermanent           = errors.New("permanent sink error")
	ErrSinkRetryAttemptTimeout = errors.New("sink attempt timeout exceeded")
	ErrWal                     = errors.New("WAL error")
	ErrShutdownTimeout         = errors.New("service shutdown timeout")
	ErrCrossValidate           = errors.New("configuration cross-validation error")
	ErrShutdown                = errors.New("shutdown requested")
)

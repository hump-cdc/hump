package errs

import "errors"

var (
	ErrCrossValidate = errors.New("configuration cross-validation error")
)

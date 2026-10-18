// Package errors holds the sentinel errors of the configuration packages.
package errors

import "errors"

var (
	// ErrCrossValidate wraps every failure of the cross-table sink validation.
	ErrCrossValidate = errors.New("configuration cross-validation error")
)

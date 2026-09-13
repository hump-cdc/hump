package sink

import "errors"

var (
	ErrUnknownSinkType = errors.New("unknown sink type")
	ErrSinkPermanent   = errors.New("permanent sink error")
)

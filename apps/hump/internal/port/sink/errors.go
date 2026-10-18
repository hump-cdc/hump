package sink

import "errors"

var (
	// ErrUnknownSinkType is returned by Sink.Process for a Definition it does not know.
	ErrUnknownSinkType = errors.New("unknown sink type")
	// ErrSinkPermanent marks a delivery failure that a retry cannot fix.
	ErrSinkPermanent = errors.New("permanent sink error")
)

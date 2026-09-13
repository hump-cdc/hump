package metricscollector

import "errors"

var (
	// ErrUnknownDuration indicates a bucket width other than the supported ones.
	ErrUnknownDuration = errors.New("unknown metrics duration")
)

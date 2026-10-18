package health

import "errors"

// ErrNotOperational is reported by a check marked unhealthy without a cause.
var ErrNotOperational = errors.New("not operational")

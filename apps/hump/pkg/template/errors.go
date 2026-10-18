package template

import "errors"

// ErrRender marks a template render failure. A render error stems from the
// template source or the shape of the data and will not resolve on retry, so
// callers can classify it as permanent.
var ErrRender = errors.New("template render")

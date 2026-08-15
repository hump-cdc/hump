package sink

import (
	sinkport "hump/internal/port/sink"
)

// CrossValidator is implemented by sink types that participate in cross-sink
// validation. Implementations inspect every sink across all tables — passed as
// one []SinkConfig group per table — and return an error wrapping
// [errs.ErrCrossValidate] when divergences are detected, or nil when the
// configuration is internally consistent.
type CrossValidator interface {
	CrossValidate(groups [][]sinkport.Definition) error
}

// CrossValidate runs each registered sink's CrossValidate against groups and
// returns the first non-nil error.
func CrossValidate(groups [][]Config) error {
	sinks := make([][]sinkport.Definition, 0, len(groups))
	for _, group := range groups {
		g := make([]sinkport.Definition, 0, len(group))
		for _, sc := range group {
			g = append(g, sc.Sink)
		}
		sinks = append(sinks, g)
	}

	for _, cv := range crossValidators {
		if err := cv.CrossValidate(sinks); err != nil {
			return err
		}
	}

	return nil
}

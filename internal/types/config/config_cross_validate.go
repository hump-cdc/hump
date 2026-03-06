package config

// CrossValidator is implemented by sink types that participate in cross-sink
// validation. Implementations inspect every sink across tables and return an
// error wrapping [errs.ErrCrossValidate] when divergences are detected, or
// nil when the configuration is internally consistent.
type CrossValidator interface {
	CrossValidate(tables []Tables) error
}

// crossValidators is the static set of sink types whose CrossValidate methods
// run during configuration validation. Each entry is a nil-receiver instance;
// implementations must not dereference the receiver.
var crossValidators = []CrossValidator{
	(*HTTPSink)(nil),
	(*KafkaSink)(nil),
	(*NatsCoreSink)(nil),
	(*NatsJetStreamSink)(nil),
	(*RabbitMQSink)(nil),
	(*RabbitMQStreamSink)(nil),
}

// CrossValidate runs each registered sink's CrossValidate against tables and
// returns the first non-nil error.
func CrossValidate(tables []Tables) error {
	for _, cv := range crossValidators {
		if err := cv.CrossValidate(tables); err != nil {
			return err
		}
	}

	return nil
}

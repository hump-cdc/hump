package errs

import (
	"fmt"
	"strings"
)

type ConfigErrorKind string

const (
	KindUnknownField    ConfigErrorKind = "unknown_field"
	KindMissingField    ConfigErrorKind = "missing_field"
	KindWrongType       ConfigErrorKind = "wrong_type"
	KindInvalidValue    ConfigErrorKind = "invalid_value"
	KindInvalidDuration ConfigErrorKind = "invalid_duration"
	KindInvalidTemplate ConfigErrorKind = "invalid_template"
	KindYAMLSyntax      ConfigErrorKind = "yaml_syntax"
	KindGeneric         ConfigErrorKind = "generic"
)

type ConfigError struct {
	Kind     ConfigErrorKind
	Message  string
	Field    string
	Value    string
	Got      string
	Want     string
	Expected string
	Line     int
	Column   int
	Path     string
	Tables   []string
	Details  string
	Cause    error
}

func (e *ConfigError) Error() string { return e.Message }

func (e *ConfigError) Unwrap() error { return e.Cause }

// Matches ErrConfigurationParse for errors.Is.
func (e *ConfigError) Is(target error) bool {
	return target == ErrConfigurationParse
}

type ConfigErrors struct {
	Errors []*ConfigError
}

func (e *ConfigErrors) Error() string {
	if len(e.Errors) == 0 {
		return "0 configuration errors"
	}
	parts := make([]string, len(e.Errors))
	for i, ce := range e.Errors {
		parts[i] = ce.Message
	}
	return fmt.Sprintf("%d configuration errors: %s", len(e.Errors), strings.Join(parts, "; "))
}

func (e *ConfigErrors) Unwrap() []error {
	out := make([]error, len(e.Errors))
	for i, ce := range e.Errors {
		out[i] = ce
	}
	return out
}

func (e *ConfigErrors) Is(target error) bool {
	return target == ErrConfigurationParse
}

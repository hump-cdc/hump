package domain

// UnavailableValue marks a column the WAL stream did not carry: an unchanged
// TOAST value with nothing to copy from the old tuple, or a non-key column of
// a key-only old tuple. It renders as an empty string everywhere, so only the
// `unavailable` template func tells it apart from one.
type UnavailableValue struct{}

// Unavailable is the value stored in a DbRecord for such a column.
var Unavailable = UnavailableValue{}

// IsUnavailable identifies the value to consumers that cannot import this
// package, such as the template function map.
func (UnavailableValue) IsUnavailable() bool { return true }

// String renders the value as an empty string.
func (UnavailableValue) String() string { return "" }

// MarshalJSON encodes the value as an empty JSON string.
func (UnavailableValue) MarshalJSON() ([]byte, error) { return []byte(`""`), nil }

// MarshalYAML encodes the value as an empty YAML string.
func (UnavailableValue) MarshalYAML() (any, error) { return "", nil }

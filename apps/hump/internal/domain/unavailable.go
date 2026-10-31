package domain

// UnavailableValue marks a column the WAL stream did not carry.
// It serializes as __hump_unavailable_value rather than an empty string or null.
type UnavailableValue struct{}

// Unavailable is the value stored in a DbRecord for such a column.
var Unavailable = UnavailableValue{}

// IsUnavailable identifies the value to consumers that cannot import this
// package, such as the template function map.
func (UnavailableValue) IsUnavailable() bool { return true }

// String renders the unavailable marker.
func (UnavailableValue) String() string { return "__hump_unavailable_value" }

// MarshalJSON encodes the unavailable marker as a JSON string.
func (UnavailableValue) MarshalJSON() ([]byte, error) {
	return []byte(`"__hump_unavailable_value"`), nil
}

// MarshalYAML encodes the unavailable marker as a YAML string.
func (UnavailableValue) MarshalYAML() (any, error) { return "__hump_unavailable_value", nil }

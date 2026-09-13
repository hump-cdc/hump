package funcs

import "reflect"

// unavailableValue is implemented by values standing in for data the source
// could not deliver. Declared as an interface so this package stays free of
// project imports.
type unavailableValue interface {
	IsUnavailable() bool
}

// isUnavailable reports whether v stands in for undelivered data. Such values
// render as an empty string and compare equal to one, so this is the only way
// to tell them apart from a genuine "".
func isUnavailable(v any) bool {
	u, ok := v.(unavailableValue)
	return ok && u.IsUnavailable()
}

// tmplUnavailable reports whether v is a value the source did not deliver,
// such as a column omitted from the replication stream.
func tmplUnavailable(v any) bool { return isUnavailable(v) }

// emptyString is what an unavailable argument stands in as during comparison.
var emptyString = reflect.ValueOf("")

// normalizeUnavailable substitutes "" for a value the source did not deliver,
// so eq and ne see the empty string it renders as rather than its own type,
// which has no basic kind and would fail the comparison.
func normalizeUnavailable(v reflect.Value) reflect.Value {
	if v.IsValid() && v.CanInterface() && isUnavailable(v.Interface()) {
		return emptyString
	}
	return v
}

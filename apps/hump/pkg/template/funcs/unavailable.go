package funcs

import "reflect"

// unavailableValue is implemented by values standing in for data the source
// could not deliver. Declared as an interface so this package stays free of
// project imports.
type unavailableValue interface {
	IsUnavailable() bool
}

// isUnavailable reports whether v stands in for undelivered data.
func isUnavailable(v any) bool {
	u, ok := v.(unavailableValue)
	return ok && u.IsUnavailable()
}

// tmplUnavailable reports whether v is a value the source did not deliver,
// such as a column omitted from the replication stream.
func tmplUnavailable(v any) bool { return isUnavailable(v) }

// emptyString is what an unavailable argument stands in as during comparison.
var emptyString = reflect.ValueOf("")

// normalizeUnavailable substitutes "" for undelivered data to preserve eq/ne semantics.
func normalizeUnavailable(v reflect.Value) reflect.Value {
	if v.IsValid() && v.CanInterface() && isUnavailable(v.Interface()) {
		return emptyString
	}
	return v
}

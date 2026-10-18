package funcs

import "fmt"

// strArg coerces a template value into a string for helpers that semantically
// take text. A nil interface returns "", a string passes through, and any
// other value is rendered with fmt.Sprint. This lets template authors pass
// missing/optional fields without the engine erroring on a type mismatch.
func strArg(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

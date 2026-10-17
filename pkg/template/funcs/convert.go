package funcs

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// tmplAtoi parses s as a base-10 integer, returning 0 on failure (matches
// sprig's silent-coerce semantics).
func tmplAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// tmplInt converts v to int via numeric or string coercion. Bools become 0/1.
// Unparseable values yield 0; values outside the int range are clamped.
func tmplInt(v any) int {
	n := tmplInt64(v)
	switch {
	case n > math.MaxInt:
		return math.MaxInt
	case n < math.MinInt:
		return math.MinInt
	}
	return int(n)
}

// tmplInt64 is like tmplInt but returns int64.
func tmplInt64(v any) int64 {
	switch x := v.(type) {
	case nil:
		return 0
	case bool:
		if x {
			return 1
		}
		return 0
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint:
		return int64(x)
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return int64(x)
	case float32:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err == nil {
			return n
		}
		f, err := strconv.ParseFloat(x, 64)
		if err == nil {
			return int64(f)
		}
		return 0
	}
	return 0
}

// tmplFloat64 converts v to float64; unparseable strings yield 0.
func tmplFloat64(v any) float64 {
	switch x := v.(type) {
	case nil:
		return 0
	case bool:
		if x {
			return 1
		}
		return 0
	case float32:
		return float64(x)
	case float64:
		return x
	case int:
		return float64(x)
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint8:
		return float64(x)
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	}
	return 0
}

// tmplToDecimal parses s as an octal-prefixed integer (Go syntax) and returns
// its decimal int64. Mirrors sprig's `todecimal`.
func tmplToDecimal(v any) int64 {
	s := fmt.Sprint(v)
	n, err := strconv.ParseInt(s, 8, 64)
	if err != nil {
		return 0
	}
	return n
}

// tmplToString stringifies any value with fmt.Sprint.
func tmplToString(v any) string { return fmt.Sprint(v) }

// tmplToStrings converts a slice/array of any element type into []string by
// stringifying each element.
func tmplToStrings(list any) []string {
	switch xs := list.(type) {
	case []string:
		return xs
	case []any:
		out := make([]string, len(xs))
		for i, v := range xs {
			out[i] = fmt.Sprint(v)
		}
		return out
	}
	return nil
}

// tmplJSON marshals a value to a JSON string. Usage:
//
//	{{.X | json}}        // compact
//	{{.X | json true}}   // indented (two spaces per level)
//
// The piped value is always the final argument; an optional leading bool
// selects pretty-printing. Marshalling errors are propagated to the template
// engine.
func tmplJSON(args ...any) (string, error) {
	var (
		prettify bool
		v        any
	)
	switch len(args) {
	case 1:
		v = args[0]
	case 2:
		p, ok := args[0].(bool)
		if !ok {
			return "", fmt.Errorf("json: first argument must be bool, got %T", args[0])
		}
		prettify = p
		v = args[1]
	default:
		return "", fmt.Errorf("json: expected 1 or 2 arguments, got %d", len(args))
	}
	var (
		b   []byte
		err error
	)
	if prettify {
		b, err = json.MarshalIndent(v, "", "  ")
	} else {
		b, err = json.Marshal(v)
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// tmplFromJSON decodes a JSON object string into map[string]any. Errors are
// surfaced to the template engine.
func tmplFromJSON(s string) (any, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

// tmplFromJSONArray decodes a JSON array string into []any.
func tmplFromJSONArray(s string) ([]any, error) {
	var v []any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

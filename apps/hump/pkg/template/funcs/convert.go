package funcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
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
		return clampUint64(uint64(x))
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return clampUint64(x)
	case float32:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		return tmplInt64(string(x))
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
	default:
		return 0
	}
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
	case json.Number:
		return tmplFloat64(string(x))
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

// tmplJSON marshals the final argument to JSON, optionally indented by a leading bool.
// NaN and ±Inf floats become null; other marshalling errors reach the template engine.
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
	b, err := marshalJSON(v, prettify)

	var unsupported *json.UnsupportedValueError
	if errors.As(err, &unsupported) {
		b, err = marshalJSON(finiteJSONValue(reflect.ValueOf(v), 0), prettify)
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// marshalJSON marshals v, indented with two spaces when prettify is set.
func marshalJSON(v any, prettify bool) ([]byte, error) {
	if prettify {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}

// finiteJSONValue returns v with every NaN or ±Inf float in its maps, slices,
// arrays, pointers and interfaces replaced by nil. A value that marshals itself,
// or one deeper than finiteJSONMaxDepth, such as a cycle, is kept as is.
func finiteJSONValue(v reflect.Value, depth int) any {
	if !v.IsValid() {
		return nil
	}
	if depth > finiteJSONMaxDepth || v.Type().Implements(jsonMarshalerType) {
		return v.Interface()
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return finiteJSONValue(v.Elem(), depth+1)
	case reflect.Map:
		if v.IsNil() || v.Type().Key().Kind() != reflect.String {
			break
		}
		out := make(map[string]any, v.Len())
		for it := v.MapRange(); it.Next(); {
			out[it.Key().String()] = finiteJSONValue(it.Value(), depth+1)
		}
		return out
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && (v.IsNil() || v.Type().Elem().Kind() == reflect.Uint8) {
			break
		}
		out := make([]any, v.Len())
		for i := range v.Len() {
			out[i] = finiteJSONValue(v.Index(i), depth+1)
		}
		return out
	default:
	}

	return v.Interface()
}

// jsonMarshalerType is the type of json.Marshaler, for finiteJSONValue.
var jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

// finiteJSONMaxDepth caps the finiteJSONValue walk, so a cyclic value reaches
// encoding/json, which reports the cycle, instead of overflowing the stack.
const finiteJSONMaxDepth = 1000

// tmplFromJSON decodes one JSON value, preserving numeric literals as json.Number.
func tmplFromJSON(s string) (any, error) {
	var v any
	if err := decodeJSON(s, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// tmplFromJSONArray decodes one JSON array or null, preserving numeric literals.
func tmplFromJSONArray(s string) ([]any, error) {
	var v []any
	if err := decodeJSON(s, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// decodeJSON decodes exactly one JSON value into target without rounding numbers.
func decodeJSON(s string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(s))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("expected a single JSON value")
	}
	return nil
}

// clampUint64 converts u to int64, clamping values above math.MaxInt64.
func clampUint64(u uint64) int64 {
	if u > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(u)
}

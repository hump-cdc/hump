package funcs

import (
	"errors"
	"reflect"
)

// tmplEmpty reports whether v is the zero value for its type. Mirrors
// Helm/sprig `empty`: nil, "", 0, false, empty slice/map/array, nil pointer.
func tmplEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String, reflect.Chan:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return rv.Complex() == 0
	case reflect.Pointer, reflect.Interface, reflect.Func:
		return rv.IsNil()
	case reflect.Struct:
		return false
	}
	return false
}

// tmplRequired returns v unchanged when non-empty; otherwise returns an error
// carrying msg. Use as a guard for fields the template author considers
// mandatory: `{{ required "name is required" .Name }}`.
func tmplRequired(msg string, v any) (any, error) {
	if tmplEmpty(v) {
		return nil, errors.New(msg)
	}
	return v, nil
}

// tmplFail unconditionally returns an error with msg. Use to abort rendering
// from a branch: `{{ if .Bad }}{{ fail "bad input" }}{{ end }}`.
func tmplFail(msg string) (string, error) {
	return "", errors.New(msg)
}

// tmplCoalesce returns the first non-empty argument, or nil when all are
// empty. Emptiness is determined by tmplEmpty.
func tmplCoalesce(args ...any) any {
	for _, a := range args {
		if !tmplEmpty(a) {
			return a
		}
	}
	return nil
}

// tmplTernary returns trueVal when cond is truthy, otherwise falseVal.
// Argument order matches sprig: trueVal, falseVal, cond.
func tmplTernary(trueVal, falseVal any, cond bool) any {
	if cond {
		return trueVal
	}
	return falseVal
}

// tmplDefault returns value when it is non-nil, otherwise def. The argument
// order is pipeline-friendly: `{{ .Field | default "fallback" }}`. Only an
// untyped-nil interface triggers the fallback — typed-nil pointers, empty
// strings, and zero numbers pass through as-is.
func tmplDefault(def string, value any) any {
	if value == nil {
		return def
	}
	return value
}

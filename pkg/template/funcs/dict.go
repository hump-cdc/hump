package funcs

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// dictArgs splits args into a dict and the remaining args, supporting both
// classic (dict-first) and pipe (dict-last) calling conventions. Returns
// ok=false when neither end is a map[string]any. Classic form wins when both
// ends are maps so existing call sites are unaffected.
func dictArgs(args []any) (map[string]any, []any, bool) {
	if len(args) == 0 {
		return nil, nil, false
	}
	if m, ok := args[0].(map[string]any); ok {
		return m, args[1:], true
	}
	if m, ok := args[len(args)-1].(map[string]any); ok {
		return m, args[:len(args)-1], true
	}
	return nil, args, false
}

// tmplDict builds a map[string]any from alternating key/value arguments.
// Odd-length input is rejected to surface mistakes early.
func tmplDict(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("dict requires an even number of arguments, got %d", len(kv))
	}
	m := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			k = fmt.Sprint(kv[i])
		}
		m[k] = kv[i+1]
	}
	return m, nil
}

// tmplGet returns d[key], or nil when missing. Accepts both
// `get $dict "key"` and `$dict | get "key"`.
func tmplGet(args ...any) (any, error) {
	d, rest, ok := dictArgs(args)
	if !ok {
		return nil, errors.New("get: first or last argument must be map[string]any")
	}
	if len(rest) != 1 {
		return nil, fmt.Errorf("get: expected one key, got %d", len(rest))
	}
	return d[fmt.Sprint(rest[0])], nil
}

// tmplSet stores value under key in d and returns d for chaining. Accepts both
// `set $dict "key" $value` and `$dict | set "key" $value`.
func tmplSet(args ...any) (map[string]any, error) {
	d, rest, ok := dictArgs(args)
	if !ok {
		return nil, errors.New("set: first or last argument must be map[string]any")
	}
	if len(rest) != 2 {
		return nil, fmt.Errorf("set: expected key and value, got %d arg(s)", len(rest))
	}
	d[fmt.Sprint(rest[0])] = rest[1]
	return d, nil
}

// tmplUnset removes key from d and returns d. Accepts both
// `unset $dict "key"` and `$dict | unset "key"`.
func tmplUnset(args ...any) (map[string]any, error) {
	d, rest, ok := dictArgs(args)
	if !ok {
		return nil, errors.New("unset: first or last argument must be map[string]any")
	}
	if len(rest) != 1 {
		return nil, fmt.Errorf("unset: expected one key, got %d", len(rest))
	}
	delete(d, fmt.Sprint(rest[0]))
	return d, nil
}

// tmplHasKey reports whether key exists in d. Accepts both
// `haskey $dict "key"` and `$dict | haskey "key"`.
func tmplHasKey(args ...any) (bool, error) {
	d, rest, ok := dictArgs(args)
	if !ok {
		return false, errors.New("haskey: first or last argument must be map[string]any")
	}
	if len(rest) != 1 {
		return false, fmt.Errorf("haskey: expected one key, got %d", len(rest))
	}
	_, ok = d[fmt.Sprint(rest[0])]
	return ok, nil
}

// tmplKeys returns the (sorted) keys of d.
func tmplKeys(d map[string]any) []string {
	ks := make([]string, 0, len(d))
	for k := range d {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// tmplValues returns the values of d in key-sorted order so output is stable.
func tmplValues(d map[string]any) []any {
	ks := tmplKeys(d)
	vs := make([]any, len(ks))
	for i, k := range ks {
		vs[i] = d[k]
	}
	return vs
}

// tmplPick returns a copy of d containing only the listed keys. Accepts both
// `pick $dict "k1" "k2"` and `$dict | pick "k1" "k2"`.
func tmplPick(args ...any) (map[string]any, error) {
	d, rest, ok := dictArgs(args)
	if !ok {
		return nil, errors.New("pick: first or last argument must be map[string]any")
	}
	out := make(map[string]any, len(rest))
	for _, k := range rest {
		ks := fmt.Sprint(k)
		if v, ok := d[ks]; ok {
			out[ks] = v
		}
	}
	return out, nil
}

// tmplPluck collects d[key] from each input dict, in argument order, when
// present.
func tmplPluck(key string, dicts ...map[string]any) []any {
	out := make([]any, 0, len(dicts))
	for _, d := range dicts {
		if v, ok := d[key]; ok {
			out = append(out, v)
		}
	}
	return out
}

// tmplDig walks nested maps using keys and returns the final value or
// fallback if any lookup misses. The argument order is sprig's:
// dig key1 key2 ... fallback dict.
func tmplDig(args ...any) (any, error) {
	if len(args) < 3 {
		return nil, errors.New("dig requires at least 3 arguments")
	}
	dict, ok := args[len(args)-1].(map[string]any)
	if !ok {
		return nil, errors.New("dig: last argument must be map[string]any")
	}
	fallback := args[len(args)-2]
	keys := args[:len(args)-2]

	var cur any = dict
	for _, k := range keys {
		ks, ok := k.(string)
		if !ok {
			ks = fmt.Sprint(k)
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return fallback, nil
		}
		v, ok := m[ks]
		if !ok {
			return fallback, nil
		}
		cur = v
	}
	return cur, nil
}

// tmplMerge does a shallow merge of src into dst, leaving existing keys in
// dst untouched. Returns dst for chaining.
func tmplMerge(dst map[string]any, srcs ...map[string]any) map[string]any {
	for _, s := range srcs {
		for k, v := range s {
			if _, exists := dst[k]; !exists {
				dst[k] = v
			}
		}
	}
	return dst
}

// tmplMergeOverwrite is tmplMerge but src values overwrite existing keys in
// dst.
func tmplMergeOverwrite(dst map[string]any, srcs ...map[string]any) map[string]any {
	for _, s := range srcs {
		for k, v := range s {
			dst[k] = v
		}
	}
	return dst
}

// tmplOmit returns a shallow copy of src with the given keys removed when
// src is a map[string]any; any other type is returned unchanged. Key matches
// are case-insensitive (strings.EqualFold), so "Password" omits "password"
// too. Accepts both `omit $dict "k1" "k2" …` and `$dict | omit "k1" "k2" …`.
func tmplOmit(args ...any) any {
	if len(args) == 0 {
		return nil
	}
	d, rest, ok := dictArgs(args)
	if !ok {
		// Non-map passthrough: return the first arg unchanged so callers
		// can chain `omit` over a heterogeneous pipeline.
		return args[0]
	}
	out := make(map[string]any, len(d))
	for k, v := range d {
		skip := false
		for _, key := range rest {
			if strings.EqualFold(k, fmt.Sprint(key)) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out[k] = v
	}
	return out
}

// tmplDeepCopy returns a deep copy of v, recursing into map[string]any and
// []any. Other types are copied by value.
func tmplDeepCopy(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = tmplDeepCopy(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = tmplDeepCopy(val)
		}
		return out
	}
	return v
}

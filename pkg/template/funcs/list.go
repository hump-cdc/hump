package funcs

import (
	"errors"
	"fmt"
	"reflect"
)

// tmplList constructs a generic []any from its arguments.
func tmplList(items ...any) []any { return items }

// toAnySlice coerces any slice/array into []any using reflection.
func toAnySlice(list any) ([]any, error) {
	if list == nil {
		return nil, nil
	}
	if xs, ok := list.([]any); ok {
		return xs, nil
	}
	rv := reflect.ValueOf(list)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected list, got %T", list)
	}
	out := make([]any, rv.Len())
	for i := range rv.Len() {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}

// tmplFirst returns the first element of list, or nil when empty.
func tmplFirst(list any) (any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, nil
	}
	return xs[0], nil
}

// tmplLast returns the last element of list, or nil when empty.
func tmplLast(list any) (any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, nil
	}
	return xs[len(xs)-1], nil
}

// tmplRest returns list without its first element.
func tmplRest(list any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	if len(xs) <= 1 {
		return []any{}, nil
	}
	return xs[1:], nil
}

// tmplInitial returns list without its last element.
func tmplInitial(list any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	if len(xs) <= 1 {
		return []any{}, nil
	}
	return xs[:len(xs)-1], nil
}

// tmplAppend appends v to list and returns the new slice.
func tmplAppend(list any, v ...any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	return append(xs, v...), nil
}

// tmplPrepend prepends v to list and returns the new slice.
func tmplPrepend(list any, v ...any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(xs)+len(v))
	out = append(out, v...)
	out = append(out, xs...)
	return out, nil
}

// tmplConcat joins the given lists into one []any.
func tmplConcat(lists ...any) ([]any, error) {
	var out []any
	for _, l := range lists {
		xs, err := toAnySlice(l)
		if err != nil {
			return nil, err
		}
		out = append(out, xs...)
	}
	return out, nil
}

// tmplReverse reverses list.
func tmplReverse(list any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(xs))
	for i, v := range xs {
		out[len(xs)-1-i] = v
	}
	return out, nil
}

// tmplUniq removes duplicate elements (first occurrence wins). Equality is
// Go's `==` operator; non-comparable elements pass through unchanged.
func tmplUniq(list any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(xs))
	seen := make(map[any]struct{}, len(xs))
	for _, v := range xs {
		if !reflect.TypeOf(v).Comparable() {
			out = append(out, v)
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out, nil
}

// tmplCompact removes empty/zero values per tmplEmpty.
func tmplCompact(list any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(xs))
	for _, v := range xs {
		if !tmplEmpty(v) {
			out = append(out, v)
		}
	}
	return out, nil
}

// tmplWithout returns list without elements equal to any of the omits.
func tmplWithout(list any, omits ...any) ([]any, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(xs))
	for _, v := range xs {
		drop := false
		for _, o := range omits {
			if reflect.DeepEqual(v, o) {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, v)
		}
	}
	return out, nil
}

// tmplHas reports whether list contains v.
func tmplHas(v, list any) (bool, error) {
	xs, err := toAnySlice(list)
	if err != nil {
		return false, err
	}
	for _, x := range xs {
		if reflect.DeepEqual(x, v) {
			return true, nil
		}
	}
	return false, nil
}

// tmplUntil returns [0, 1, ..., n-1]. Negative n yields a descending sequence.
func tmplUntil(n int) []int {
	if n == 0 {
		return []int{}
	}
	if n > 0 {
		out := make([]int, n)
		for i := range n {
			out[i] = i
		}
		return out
	}
	out := make([]int, -n)
	for i := 0; i > n; i-- {
		out[-i] = i
	}
	return out
}

// tmplUntilStep returns [start, start+step, ...] up to (but not including) end.
func tmplUntilStep(start, end, step int) []int {
	var out []int
	if step == 0 {
		return out
	}
	if step > 0 {
		for i := start; i < end; i += step {
			out = append(out, i)
		}
	} else {
		for i := start; i > end; i += step {
			out = append(out, i)
		}
	}
	return out
}

// tmplSeq emits a Bash-style numeric sequence. With one arg it's [1..n];
// with two it's [a..b]; with three it's [start, start+step, ...]. Decreasing
// ranges work when the implied step would be negative.
func tmplSeq(args ...int) []int {
	switch len(args) {
	case 1:
		return tmplSeqRange(1, args[0], 1)
	case 2:
		step := 1
		if args[0] > args[1] {
			step = -1
		}
		return tmplSeqRange(args[0], args[1], step)
	case 3:
		return tmplSeqRange(args[0], args[2], args[1])
	}
	return nil
}

func tmplSeqRange(start, end, step int) []int {
	var out []int
	if step == 0 {
		return out
	}
	if step > 0 {
		for i := start; i <= end; i += step {
			out = append(out, i)
		}
	} else {
		for i := start; i >= end; i += step {
			out = append(out, i)
		}
	}
	return out
}

// tmplChunk splits list into groups of size. The final group may be smaller.
func tmplChunk(size int, list any) ([][]any, error) {
	if size <= 0 {
		return nil, errors.New("chunk size must be positive")
	}
	xs, err := toAnySlice(list)
	if err != nil {
		return nil, err
	}
	out := make([][]any, 0, (len(xs)+size-1)/size)
	for i := 0; i < len(xs); i += size {
		end := i + size
		if end > len(xs) {
			end = len(xs)
		}
		out = append(out, xs[i:end])
	}
	return out, nil
}

package funcs

import (
	"errors"
	"fmt"
	"reflect"
)

// Builtins mirrors text/template/funcs.go builtins() so callers can use it as
// the canonical, complete set of functions available to parsed templates.

var errorType = reflect.TypeFor[error]()

func goodFunc(name string, typ reflect.Type) error {
	switch numOut := typ.NumOut(); {
	case numOut == 1:
		return nil
	case numOut == 2 && typ.Out(1) == errorType:
		return nil
	case numOut == 2:
		return fmt.Errorf("invalid function signature for %s: second return value should be error; is %s", name, typ.Out(1))
	default:
		return fmt.Errorf("function %s has %d return values; should be 1 tmplOr 2", name, typ.NumOut())
	}
}

func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}

func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem()
}

func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	case reflect.Struct:
		return typ == reflect.TypeFor[reflect.Value]()
	}
	return false
}

func isTrue(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		return false, true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Pointer, reflect.UnsafePointer, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true
	}
	return truth, true
}

func truth(arg reflect.Value) bool {
	t, _ := isTrue(indirectInterface(arg))
	return t
}

func intLike(typ reflect.Kind) bool {
	switch typ {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return false
}

func prepareArg(value reflect.Value, argType reflect.Type) (reflect.Value, error) {
	if !value.IsValid() {
		if !canBeNil(argType) {
			return reflect.Value{}, fmt.Errorf("value is nil; should be of type %s", argType)
		}
		value = reflect.Zero(argType)
	}
	if value.Type().AssignableTo(argType) {
		return value, nil
	}
	if intLike(value.Kind()) && intLike(argType.Kind()) && value.Type().ConvertibleTo(argType) {
		value = value.Convert(argType)
		return value, nil
	}
	return reflect.Value{}, fmt.Errorf("value has type %s; should be %s", value.Type(), argType)
}

func indexArg(tmplIndex reflect.Value, cap int) (int, error) {
	var x int64
	switch tmplIndex.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x = tmplIndex.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		x = int64(tmplIndex.Uint())
	case reflect.Invalid:
		return 0, fmt.Errorf("cannot tmplIndex tmplSlice/array with nil")
	default:
		return 0, fmt.Errorf("cannot tmplIndex tmplSlice/array with type %s", tmplIndex.Type())
	}
	if x < 0 || int(x) < 0 || int(x) > cap {
		return 0, fmt.Errorf("tmplIndex out of range: %d", x)
	}
	return int(x), nil
}

// tmplIndex returns the result of indexing its first argument by the following
// arguments. Thus "tmplIndex x 1 2 3" is, in Go syntax, x[1][2][3].
func tmplIndex(item reflect.Value, indexes ...reflect.Value) (reflect.Value, error) {
	item = indirectInterface(item)
	if !item.IsValid() {
		return reflect.Value{}, fmt.Errorf("tmplIndex of untyped nil")
	}
	for _, tmplIndex := range indexes {
		tmplIndex = indirectInterface(tmplIndex)
		var isNil bool
		if item, isNil = indirect(item); isNil {
			return reflect.Value{}, fmt.Errorf("tmplIndex of nil pointer")
		}
		switch item.Kind() {
		case reflect.Array, reflect.Slice, reflect.String:
			x, err := indexArg(tmplIndex, item.Len())
			if err != nil {
				return reflect.Value{}, err
			}
			item = item.Index(x)
		case reflect.Map:
			tmplIndex, err := prepareArg(tmplIndex, item.Type().Key())
			if err != nil {
				return reflect.Value{}, err
			}
			if x := item.MapIndex(tmplIndex); x.IsValid() {
				item = x
			} else {
				item = reflect.Zero(item.Type().Elem())
			}
		default:
			return reflect.Value{}, fmt.Errorf("can't tmplIndex item of type %s", item.Type())
		}
	}
	return item, nil
}

// tmplSlice returns the result of slicing its first argument by the remaining
// arguments. Thus "tmplSlice x 1 2" is, in Go syntax, x[1:2].
func tmplSlice(item reflect.Value, indexes ...reflect.Value) (reflect.Value, error) {
	item = indirectInterface(item)
	if !item.IsValid() {
		return reflect.Value{}, fmt.Errorf("tmplSlice of untyped nil")
	}
	var isNil bool
	if item, isNil = indirect(item); isNil {
		return reflect.Value{}, fmt.Errorf("tmplSlice of nil pointer")
	}
	if len(indexes) > 3 {
		return reflect.Value{}, fmt.Errorf("too many tmplSlice indexes: %d", len(indexes))
	}
	var cap int
	switch item.Kind() {
	case reflect.String:
		if len(indexes) == 3 {
			return reflect.Value{}, fmt.Errorf("cannot 3-tmplIndex tmplSlice a string")
		}
		cap = item.Len()
	case reflect.Array, reflect.Slice:
		cap = item.Cap()
	default:
		return reflect.Value{}, fmt.Errorf("can't tmplSlice item of type %s", item.Type())
	}
	idx := [3]int{0, item.Len()}
	for i, tmplIndex := range indexes {
		x, err := indexArg(tmplIndex, cap)
		if err != nil {
			return reflect.Value{}, err
		}
		idx[i] = x
	}
	if idx[0] > idx[1] {
		return reflect.Value{}, fmt.Errorf("invalid tmplSlice tmplIndex: %d > %d", idx[0], idx[1])
	}
	if len(indexes) < 3 {
		return item.Slice(idx[0], idx[1]), nil
	}
	if idx[1] > idx[2] {
		return reflect.Value{}, fmt.Errorf("invalid tmplSlice tmplIndex: %d > %d", idx[1], idx[2])
	}
	return item.Slice3(idx[0], idx[1], idx[2]), nil
}

// tmplLength returns the tmplLength of the item, with an error if it has no defined tmplLength.
func tmplLength(item reflect.Value) (int, error) {
	item, isNil := indirect(item)
	if isNil {
		return 0, fmt.Errorf("len of nil pointer")
	}
	switch item.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return item.Len(), nil
	}
	return 0, fmt.Errorf("len of type %s", item.Type())
}

// tmplCall returns the result of evaluating the first argument as a function.
// The function must return 1 result, tmplOr 2 results, the second of which is an error.
func tmplCall(fn reflect.Value, args ...reflect.Value) (reflect.Value, error) {
	const name = "tmplCall"
	fn = indirectInterface(fn)
	if !fn.IsValid() {
		return reflect.Value{}, fmt.Errorf("tmplCall of nil")
	}
	typ := fn.Type()
	if typ.Kind() != reflect.Func {
		return reflect.Value{}, fmt.Errorf("non-function %s of type %s", name, typ)
	}

	if err := goodFunc(name, typ); err != nil {
		return reflect.Value{}, err
	}
	numIn := typ.NumIn()
	var dddType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return reflect.Value{}, fmt.Errorf("wrong number of args for %s: got %d want at least %d", name, len(args), numIn-1)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return reflect.Value{}, fmt.Errorf("wrong number of args for %s: got %d want %d", name, len(args), numIn)
		}
	}
	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		arg = indirectInterface(arg)
		argType := dddType
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		}
		var err error
		if argv[i], err = prepareArg(arg, argType); err != nil {
			return reflect.Value{}, fmt.Errorf("arg %d: %w", i, err)
		}
	}
	return safeCall(fn, argv)
}

func safeCall(fun reflect.Value, args []reflect.Value) (val reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	ret := fun.Call(args)
	if len(ret) == 2 && !ret[1].IsNil() {
		return ret[0], ret[1].Interface().(error)
	}
	return ret[0], nil
}

// tmplAnd returns the first false argument, tmplOr the last argument when all are true.
func tmplAnd(arg0 reflect.Value, args ...reflect.Value) reflect.Value {
	if !truth(arg0) {
		return arg0
	}
	for i := range args {
		arg0 = args[i]
		if !truth(arg0) {
			break
		}
	}
	return arg0
}

// tmplOr returns the first true argument, tmplOr the last argument when all are false.
func tmplOr(arg0 reflect.Value, args ...reflect.Value) reflect.Value {
	if truth(arg0) {
		return arg0
	}
	for i := range args {
		arg0 = args[i]
		if truth(arg0) {
			break
		}
	}
	return arg0
}

func tmplNot(arg reflect.Value) bool {
	return !truth(arg)
}

var (
	errBadComparisonType = errors.New("invalid type for comparison")
	errNoComparison      = errors.New("missing argument for comparison")
)

type kind int

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	stringKind
	uintKind
)

func basicKind(v reflect.Value) (kind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	}
	return invalidKind, errBadComparisonType
}

func isNilValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func canCompare(v1, v2 reflect.Value) bool {
	k1 := v1.Kind()
	k2 := v2.Kind()
	if k1 == k2 {
		return true
	}
	return k1 == reflect.Invalid || k2 == reflect.Invalid
}

// tmplEq evaluates the comparison a == b || a == c || ...
func tmplEq(arg1 reflect.Value, arg2 ...reflect.Value) (bool, error) {
	arg1 = normalizeUnavailable(indirectInterface(arg1))
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	k1, _ := basicKind(arg1)
	for _, arg := range arg2 {
		arg = normalizeUnavailable(indirectInterface(arg))
		k2, _ := basicKind(arg)
		t := false
		if k1 != k2 {
			switch {
			case k1 == intKind && k2 == uintKind:
				t = arg1.Int() >= 0 && uint64(arg1.Int()) == arg.Uint()
			case k1 == uintKind && k2 == intKind:
				t = arg.Int() >= 0 && arg1.Uint() == uint64(arg.Int())
			default:
				if arg1.IsValid() && arg.IsValid() {
					return false, fmt.Errorf("incompatible types for comparison: %v tmplAnd %v", arg1.Type(), arg.Type())
				}
			}
		} else {
			switch k1 {
			case boolKind:
				t = arg1.Bool() == arg.Bool()
			case complexKind:
				t = arg1.Complex() == arg.Complex()
			case floatKind:
				t = arg1.Float() == arg.Float()
			case intKind:
				t = arg1.Int() == arg.Int()
			case stringKind:
				t = arg1.String() == arg.String()
			case uintKind:
				t = arg1.Uint() == arg.Uint()
			default:
				if !canCompare(arg1, arg) {
					return false, fmt.Errorf("non-comparable types %s: %v, %s: %v", arg1, arg1.Type(), arg.Type(), arg)
				}
				if isNilValue(arg1) || isNilValue(arg) {
					t = isNilValue(arg) == isNilValue(arg1)
				} else {
					if !arg.Type().Comparable() {
						return false, fmt.Errorf("non-comparable type %s: %v", arg, arg.Type())
					}
					t = arg1.Interface() == arg.Interface()
				}
			}
		}
		if t {
			return true, nil
		}
	}
	return false, nil
}

func tmplNe(arg1, arg2 reflect.Value) (bool, error) {
	equal, err := tmplEq(arg1, arg2)
	return !equal, err
}

func tmplLt(arg1, arg2 reflect.Value) (bool, error) {
	arg1 = indirectInterface(arg1)
	k1, err := basicKind(arg1)
	if err != nil {
		return false, err
	}
	arg2 = indirectInterface(arg2)
	k2, err := basicKind(arg2)
	if err != nil {
		return false, err
	}
	t := false
	if k1 != k2 {
		switch {
		case k1 == intKind && k2 == uintKind:
			t = arg1.Int() < 0 || uint64(arg1.Int()) < arg2.Uint()
		case k1 == uintKind && k2 == intKind:
			t = arg2.Int() >= 0 && arg1.Uint() < uint64(arg2.Int())
		default:
			return false, fmt.Errorf("incompatible types for comparison: %v tmplAnd %v", arg1.Type(), arg2.Type())
		}
	} else {
		switch k1 {
		case boolKind, complexKind:
			return false, errBadComparisonType
		case floatKind:
			t = arg1.Float() < arg2.Float()
		case intKind:
			t = arg1.Int() < arg2.Int()
		case stringKind:
			t = arg1.String() < arg2.String()
		case uintKind:
			t = arg1.Uint() < arg2.Uint()
		}
	}
	return t, nil
}

func tmplLe(arg1, arg2 reflect.Value) (bool, error) {
	lessThan, err := tmplLt(arg1, arg2)
	if lessThan || err != nil {
		return lessThan, err
	}
	return tmplEq(arg1, arg2)
}

func tmplGt(arg1, arg2 reflect.Value) (bool, error) {
	lessOrEqual, err := tmplLe(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessOrEqual, nil
}

func tmplGe(arg1, arg2 reflect.Value) (bool, error) {
	lessThan, err := tmplLt(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessThan, nil
}

package funcs

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestBuiltinTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"and last when all true", `{{ and .A .B .C }}`, map[string]any{"A": "x", "B": "y", "C": "z"}, "z"},
		{"and short-circuit empty", `{{ and .A .B }}`, map[string]any{"A": "", "B": "y"}, ""},
		{"or first true", `{{ or .A .B .C }}`, map[string]any{"A": "", "B": "y", "C": "z"}, "y"},
		{"or all empty", `{{ or .A .B }}`, map[string]any{"A": "", "B": ""}, ""},
		{"not false → true", `{{ not .Value }}`, V(false), "true"},
		{"not non-empty → false", `{{ not .Value }}`, V("x"), "false"},

		{"eq", `{{ eq 1 1 }}`, nil, "true"},
		{"eq many false", `{{ eq 1 2 }}`, nil, "false"},
		{"eq many true", `{{ eq "a" "b" "a" }}`, nil, "true"},
		{"ne", `{{ ne 1 2 }}`, nil, "true"},
		{"lt", `{{ lt 1 2 }}`, nil, "true"},
		{"le eq", `{{ le 2 2 }}`, nil, "true"},
		{"gt", `{{ gt 3 2 }}`, nil, "true"},
		{"ge eq", `{{ ge 2 2 }}`, nil, "true"},

		{"len slice", `{{ len .Value }}`, V([]int{1, 2, 3}), "3"},
		{"len string", `{{ len "hello" }}`, nil, "5"},

		{"index slice", `{{ index .Value 1 }}`, V([]string{"a", "b", "c"}), "b"},
		{"index nested map",
			`{{ index .Value "a" "b" }}`,
			V(map[string]any{"a": map[string]any{"b": "deep"}}),
			"deep"},

		{"slice subrange", `{{ slice .Value 1 3 }}`, V([]int{1, 2, 3, 4}), "[2 3]"},
		{"slice string", `{{ slice "hello" 1 4 }}`, nil, "ell"},

		{"call", `{{ call .Add 2 3 }}`, map[string]any{
			"Add": func(a, b int) int { return a + b },
		}, "5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, tc.data)
			if got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

func TestBuiltinTemplateErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"call two results without error", `{{ call .Value }}`, V(func() (int, string) { return 0, "" }),
			"second return value should be error"},
		{"call three results", `{{ call .Value }}`, V(func() (int, int, int) { return 0, 0, 0 }), "should be 1 tmplOr 2"},
		{"call error result", `{{ call .Value }}`, V(func() (int, error) { return 0, errors.New("boom") }), "boom"},
		{"index nil pointer", `{{ index .Value 0 }}`, V((*[]int)(nil)), "tmplIndex of nil pointer"},
		{"len nil pointer", `{{ len .Value }}`, V((*[]int)(nil)), "len of nil pointer"},
		{"eq no args", `{{ eq 1 }}`, nil, "missing argument for comparison"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := render(t, tc.src, tc.data)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%s error = %v, want it to contain %q", tc.src, err, tc.want)
			}
		})
	}
}

func TestIndirectInterface(t *testing.T) {
	nilErr := reflect.ValueOf([]error{nil}).Index(0)
	if got := indirectInterface(nilErr); got.IsValid() {
		t.Fatalf("indirectInterface(nil interface) = %v, want invalid", got)
	}

	set := reflect.ValueOf([]error{errors.New("x")}).Index(0)
	if got := indirectInterface(set); got.Kind() != reflect.Pointer {
		t.Fatalf("indirectInterface(set interface) kind = %v, want pointer", got.Kind())
	}
}

func TestCanBeNil(t *testing.T) {
	cases := []struct {
		typ  reflect.Type
		want bool
	}{
		{reflect.TypeFor[chan int](), true},
		{reflect.TypeFor[[]int](), true},
		{reflect.TypeFor[reflect.Value](), true},
		{reflect.TypeFor[struct{}](), false},
		{reflect.TypeFor[int](), false},
	}
	for _, tc := range cases {
		if got := canBeNil(tc.typ); got != tc.want {
			t.Errorf("canBeNil(%v) = %v, want %v", tc.typ, got, tc.want)
		}
	}
}

func TestIsTrue(t *testing.T) {
	var nilPtr *int
	cases := []struct {
		name string
		val  reflect.Value
		want bool
	}{
		{"invalid", reflect.Value{}, false},
		{"empty string", reflect.ValueOf(""), false},
		{"true", reflect.ValueOf(true), true},
		{"complex zero", reflect.ValueOf(complex(0, 0)), false},
		{"complex", reflect.ValueOf(complex(1, 0)), true},
		{"nil pointer", reflect.ValueOf(nilPtr), false},
		{"func", reflect.ValueOf(func() {}), true},
		{"int", reflect.ValueOf(-1), true},
		{"float zero", reflect.ValueOf(0.0), false},
		{"uint", reflect.ValueOf(uint8(1)), true},
		{"struct", reflect.ValueOf(struct{}{}), true},
	}
	for _, tc := range cases {
		if got := isTrue(tc.val); got != tc.want {
			t.Errorf("isTrue(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIntLike(t *testing.T) {
	cases := []struct {
		kind reflect.Kind
		want bool
	}{
		{reflect.Int8, true},
		{reflect.Uintptr, true},
		{reflect.Float64, false},
	}
	for _, tc := range cases {
		if got := intLike(tc.kind); got != tc.want {
			t.Errorf("intLike(%v) = %v, want %v", tc.kind, got, tc.want)
		}
	}
}

func TestPrepareArg(t *testing.T) {
	cases := []struct {
		name    string
		value   reflect.Value
		argType reflect.Type
		want    any
		wantErr string
	}{
		{"nil into non-nilable", reflect.Value{}, reflect.TypeFor[int](), nil, "value is nil; should be of type int"},
		{"nil into slice", reflect.Value{}, reflect.TypeFor[[]int](), []int(nil), ""},
		{"assignable", reflect.ValueOf("a"), reflect.TypeFor[string](), "a", ""},
		{"int converted", reflect.ValueOf(int8(3)), reflect.TypeFor[int64](), int64(3), ""},
		{"mismatch", reflect.ValueOf("a"), reflect.TypeFor[int](), nil, "value has type string; should be int"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := prepareArg(tc.value, tc.argType)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("prepareArg error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepareArg: %v", err)
			}
			if !reflect.DeepEqual(got.Interface(), tc.want) {
				t.Fatalf("prepareArg = %#v, want %#v", got.Interface(), tc.want)
			}
		})
	}
}

func TestIndexArg(t *testing.T) {
	cases := []struct {
		name    string
		index   reflect.Value
		want    int
		wantErr string
	}{
		{"int", reflect.ValueOf(2), 2, ""},
		{"uint", reflect.ValueOf(uint(1)), 1, ""},
		{"uint over int64", reflect.ValueOf(uint64(math.MaxUint64)), 0, "tmplIndex out of range"},
		{"nil", reflect.Value{}, 0, "cannot tmplIndex tmplSlice/array with nil"},
		{"string", reflect.ValueOf("x"), 0, "cannot tmplIndex tmplSlice/array with type string"},
		{"negative", reflect.ValueOf(-1), 0, "tmplIndex out of range: -1"},
		{"over capacity", reflect.ValueOf(4), 0, "tmplIndex out of range: 4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := indexArg(tc.index, 3)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("indexArg error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("indexArg = %d, %v, want %d", got, err, tc.want)
			}
		})
	}
}

func TestTmplIndex(t *testing.T) {
	cases := []struct {
		name    string
		item    reflect.Value
		indexes []reflect.Value
		want    any
		wantErr string
	}{
		{"untyped nil", reflect.Value{}, nil, nil, "tmplIndex of untyped nil"},
		{"bad index", reflect.ValueOf([]int{1}), []reflect.Value{reflect.ValueOf("x")}, nil, "with type string"},
		{"bad map key", reflect.ValueOf(map[int]string{1: "a"}), []reflect.Value{reflect.ValueOf("a")}, nil,
			"value has type string; should be int"},
		{"missing map key", reflect.ValueOf(map[string]int{"a": 1}), []reflect.Value{reflect.ValueOf("b")}, 0, ""},
		{"not indexable", reflect.ValueOf(5), []reflect.Value{reflect.ValueOf(0)}, nil, "can't tmplIndex item of type int"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tmplIndex(tc.item, tc.indexes...)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("tmplIndex error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got.Interface() != tc.want {
				t.Fatalf("tmplIndex = %v, %v, want %v", got, err, tc.want)
			}
		})
	}
}

func TestTmplSlice(t *testing.T) {
	ints := reflect.ValueOf([]int{1, 2, 3, 4})
	i := func(n int) reflect.Value { return reflect.ValueOf(n) }
	cases := []struct {
		name    string
		item    reflect.Value
		indexes []reflect.Value
		want    any
		wantErr string
	}{
		{"untyped nil", reflect.Value{}, nil, nil, "tmplSlice of untyped nil"},
		{"nil pointer", reflect.ValueOf((*[]int)(nil)), nil, nil, "tmplSlice of nil pointer"},
		{"too many indexes", ints, []reflect.Value{i(0), i(1), i(2), i(3)}, nil, "too many tmplSlice indexes: 4"},
		{"3-index string", reflect.ValueOf("abc"), []reflect.Value{i(0), i(1), i(2)}, nil,
			"cannot 3-tmplIndex tmplSlice a string"},
		{"not sliceable", reflect.ValueOf(5), nil, nil, "can't tmplSlice item of type int"},
		{"bad index", ints, []reflect.Value{reflect.ValueOf("x")}, nil, "with type string"},
		{"low over high", ints, []reflect.Value{i(3), i(1)}, nil, "invalid tmplSlice tmplIndex: 3 > 1"},
		{"high over max", ints, []reflect.Value{i(0), i(3), i(2)}, nil, "invalid tmplSlice tmplIndex: 3 > 2"},
		{"3-index", ints, []reflect.Value{i(1), i(2), i(3)}, []int{2}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tmplSlice(tc.item, tc.indexes...)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("tmplSlice error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got.Interface(), tc.want) {
				t.Fatalf("tmplSlice = %v, %v, want %v", got, err, tc.want)
			}
		})
	}
}

func TestTmplLengthOfUnsizedType(t *testing.T) {
	if _, err := tmplLength(reflect.ValueOf(5)); err == nil || err.Error() != "len of type int" {
		t.Fatalf("tmplLength(5) error = %v, want len of type int", err)
	}
}

func TestTmplCall(t *testing.T) {
	join := reflect.ValueOf(func(sep string, parts ...string) string { return strings.Join(parts, sep) })
	s := func(v string) reflect.Value { return reflect.ValueOf(v) }
	cases := []struct {
		name    string
		fn      reflect.Value
		args    []reflect.Value
		want    any
		wantErr string
	}{
		{"nil", reflect.Value{}, nil, nil, "tmplCall of nil"},
		{"non-function", reflect.ValueOf(5), nil, nil, "non-function tmplCall of type int"},
		{"bad signature", reflect.ValueOf(func() {}), nil, nil, "has 0 return values"},
		{"variadic too few", join, nil, nil, "got 0 want at least 1"},
		{"variadic", join, []reflect.Value{s("-"), s("a"), s("b")}, "a-b", ""},
		{"wrong arity", reflect.ValueOf(func(int) int { return 0 }), nil, nil, "got 0 want 1"},
		{"bad arg", reflect.ValueOf(func(int) int { return 0 }), []reflect.Value{s("x")}, nil,
			"arg 0: value has type string; should be int"},
		{"panics with error", reflect.ValueOf(func() int { panic(errors.New("err panic")) }), nil, nil, "err panic"},
		{"panics with value", reflect.ValueOf(func() int { panic("str panic") }), nil, nil, "str panic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tmplCall(tc.fn, tc.args...)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("tmplCall error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got.Interface() != tc.want {
				t.Fatalf("tmplCall = %v, %v, want %v", got, err, tc.want)
			}
		})
	}
}

func TestTmplAndOr(t *testing.T) {
	one, zero, two := reflect.ValueOf(1), reflect.ValueOf(0), reflect.ValueOf(2)

	if got := tmplAnd(one, zero, two); got.Interface() != 0 {
		t.Errorf("and 1 0 2 = %v, want 0", got)
	}
	if got := tmplOr(one, two); got.Interface() != 1 {
		t.Errorf("or 1 2 = %v, want 1", got)
	}
}

func TestBasicKind(t *testing.T) {
	cases := []struct {
		val  any
		want kind
	}{
		{true, boolKind},
		{1, intKind},
		{uint(1), uintKind},
		{1.5, floatKind},
		{complex(1, 1), complexKind},
		{"s", stringKind},
	}
	for _, tc := range cases {
		if got, err := basicKind(reflect.ValueOf(tc.val)); err != nil || got != tc.want {
			t.Errorf("basicKind(%#v) = %v, %v, want %v", tc.val, got, err, tc.want)
		}
	}

	if _, err := basicKind(reflect.ValueOf(struct{}{})); !errors.Is(err, errBadComparisonType) {
		t.Errorf("basicKind(struct) error = %v, want errBadComparisonType", err)
	}
}

func TestIsNilValue(t *testing.T) {
	cases := []struct {
		name string
		val  reflect.Value
		want bool
	}{
		{"invalid", reflect.Value{}, true},
		{"nil slice", reflect.ValueOf([]int(nil)), true},
		{"map", reflect.ValueOf(map[string]int{}), false},
		{"struct", reflect.ValueOf(struct{}{}), false},
	}
	for _, tc := range cases {
		if got := isNilValue(tc.val); got != tc.want {
			t.Errorf("isNilValue(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCanCompare(t *testing.T) {
	cases := []struct {
		name   string
		v1, v2 reflect.Value
		want   bool
	}{
		{"same kind", reflect.ValueOf([]int{}), reflect.ValueOf([]int{}), true},
		{"one invalid", reflect.Value{}, reflect.ValueOf([]int{}), true},
		{"different kinds", reflect.ValueOf([]int{}), reflect.ValueOf(map[int]int{}), false},
	}
	for _, tc := range cases {
		if got := canCompare(tc.v1, tc.v2); got != tc.want {
			t.Errorf("canCompare(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestTmplEq(t *testing.T) {
	type pair struct{ A, B int }
	cases := []struct {
		name    string
		a, b    any
		want    bool
		wantErr string
	}{
		{"int and uint", 1, uint(1), true, ""},
		{"uint and int", uint(2), 2, true, ""},
		{"negative int and uint", -1, uint(1), false, ""},
		{"int and string", 1, "a", false, "incompatible types for comparison: int tmplAnd string"},
		{"int and nil", 1, nil, false, ""},
		{"bool", true, true, true, ""},
		{"complex", complex(1, 2), complex(1, 2), true, ""},
		{"float", 1.5, 2.5, false, ""},
		{"uint", uint(3), uint(3), true, ""},
		{"nil and nil", nil, nil, true, ""},
		{"nil and slice", nil, []int{1}, false, ""},
		{"structs", pair{1, 2}, pair{1, 2}, true, ""},
		{"slices", []int{1}, []int{1}, false, "non-comparable type"},
		{"slice and map", []int{1}, map[int]int{}, false, "non-comparable types"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tmplEq(reflect.ValueOf(tc.a), reflect.ValueOf(tc.b))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("tmplEq error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("tmplEq(%#v, %#v) = %v, %v, want %v", tc.a, tc.b, got, err, tc.want)
			}
		})
	}
}

func TestTmplOrdering(t *testing.T) {
	cases := []struct {
		name           string
		a, b           any
		lt, le, gt, ge bool
		wantErr        string
	}{
		{"int lt uint", 1, uint(2), true, true, false, false, ""},
		{"uint lt int", uint(2), 1, false, false, true, true, ""},
		{"floats", 1.5, 1.5, false, true, false, true, ""},
		{"strings", "a", "b", true, true, false, false, ""},
		{"uints", uint(3), uint(1), false, false, true, true, ""},
		{"bools", true, false, false, false, false, false, "invalid type for comparison"},
		{"struct first", struct{}{}, 1, false, false, false, false, "invalid type for comparison"},
		{"struct second", 1, struct{}{}, false, false, false, false, "invalid type for comparison"},
		{"int and string", 1, "a", false, false, false, false, "incompatible types for comparison"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := reflect.ValueOf(tc.a), reflect.ValueOf(tc.b)
			checks := []struct {
				op   string
				fn   func(a, b reflect.Value) (bool, error)
				want bool
			}{
				{"lt", tmplLt, tc.lt},
				{"le", tmplLe, tc.le},
				{"gt", tmplGt, tc.gt},
				{"ge", tmplGe, tc.ge},
			}
			for _, c := range checks {
				got, err := c.fn(a, b)
				if tc.wantErr != "" {
					if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("%s error = %v, want %q", c.op, err, tc.wantErr)
					}
					continue
				}
				if err != nil || got != c.want {
					t.Errorf("%s(%#v, %#v) = %v, %v, want %v", c.op, tc.a, tc.b, got, err, c.want)
				}
			}
		})
	}
}

func TestMixedSignComparisons(t *testing.T) {
	cases := []struct {
		name string
		i    int64
		u    uint64
		eq   bool
		iLtU bool
		uLtI bool
	}{
		{"equal", 5, 5, true, false, false},
		{"int smaller", 1, 5, false, true, false},
		{"uint smaller", 5, 1, false, false, true},
		{"negative int", -1, 0, false, true, false},
		{"huge uint", math.MaxInt64, math.MaxUint64, false, true, false},
	}
	for _, tc := range cases {
		if got := intEqUint(tc.i, tc.u); got != tc.eq {
			t.Errorf("%s: intEqUint = %v, want %v", tc.name, got, tc.eq)
		}
		if got := intLtUint(tc.i, tc.u); got != tc.iLtU {
			t.Errorf("%s: intLtUint = %v, want %v", tc.name, got, tc.iLtU)
		}
		if got := uintLtInt(tc.u, tc.i); got != tc.uLtI {
			t.Errorf("%s: uintLtInt = %v, want %v", tc.name, got, tc.uLtI)
		}
	}
}

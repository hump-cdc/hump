package funcs

import (
	"math"
	"strconv"
	"testing"
)

func TestConvertTemplates(t *testing.T) {
	cases := []struct {
		name string
		src  string
		in   any
		want string
	}{
		{"atoi", `{{ atoi .Value }}`, "42", "42"},
		{"atoi bad", `{{ atoi .Value }}`, "abc", "0"},
		{"int from string", `{{ int .Value }}`, "12", "12"},
		{"int from big string", `{{ int .Value }}`, "9223372036854775807", strconv.Itoa(math.MaxInt)},
		{"int from float", `{{ int .Value }}`, 3.7, "3"},
		{"int from bool", `{{ int .Value }}`, true, "1"},
		{"int nil", `{{ int .Value }}`, nil, "0"},
		{"int64 from uint", `{{ int64 .Value }}`, uint(7), "7"},
		{"float64 from int", `{{ float64 .Value }}`, 5, "5"},
		{"float64 from string", `{{ float64 .Value }}`, "2.5", "2.5"},
		{"float64 bad", `{{ float64 .Value }}`, "x", "0"},
		{"todecimal", `{{ todecimal .Value }}`, "777", "511"},
		{"tostring int", `{{ tostring .Value }}`, 42, "42"},
		{"tostring nil", `{{ tostring .Value }}`, nil, "<nil>"},
		{"tostrings []string", `{{ tostrings .Value }}`, []string{"a", "b"}, "[a b]"},
		{"tostrings []any", `{{ tostrings .Value }}`, []any{1, "x", true}, "[1 x true]"},
		{"tostrings other", `{{ tostrings .Value }}`, 5, "[]"},
		{"json compact", `{{ json false .Value }}`, map[string]any{"a": 1}, `{"a":1}`},
		{"json pretty", `{{ json true .Value }}`, map[string]any{"a": 1}, "{\n  \"a\": 1\n}"},
		{"json default pipe", `{{ .Value | json }}`, map[string]any{"a": 1}, `{"a":1}`},
		{"json pretty pipe", `{{ .Value | json true }}`, map[string]any{"a": 1}, "{\n  \"a\": 1\n}"},
		{"fromjson get", `{{ index (fromjson .Value) "a" }}`, `{"a":1}`, "1"},
		{"fromjsonarray", `{{ index (fromjsonarray .Value) 1 }}`, `[1,"x",true]`, "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, V(tc.in))
			if got != tc.want {
				t.Fatalf("%s with %v = %q, want %q", tc.src, tc.in, got, tc.want)
			}
		})
	}
}

func TestConvertTemplate_Errors(t *testing.T) {
	if _, err := render(t, `{{ json false .Value }}`, V(make(chan int))); err == nil {
		t.Fatalf("expected json marshal error")
	}
	if _, err := render(t, `{{ fromjson .Value }}`, V("{bad}")); err == nil {
		t.Fatalf("expected fromjson error")
	}
	if _, err := render(t, `{{ fromjsonarray .Value }}`, V("[bad]")); err == nil {
		t.Fatalf("expected fromjsonarray error")
	}
}

// TestInt64Template exercises the type branches of the int64 conversion through
// template rendering.
func TestInt64Template(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "0"},
		{"bool true", true, "1"},
		{"bool false", false, "0"},
		{"int", int(5), "5"},
		{"int8", int8(6), "6"},
		{"int16", int16(7), "7"},
		{"int32", int32(8), "8"},
		{"int64", int64(9), "9"},
		{"uint", uint(10), "10"},
		{"uint8", uint8(11), "11"},
		{"uint16", uint16(12), "12"},
		{"uint32", uint32(13), "13"},
		{"uint64", uint64(14), "14"},
		{"float32", float32(15.9), "15"},
		{"float64", float64(16.9), "16"},
		{"string int", "17", "17"},
		{"string float", "18.9", "18"},
		{"string bad", "nope", "0"},
		{"unsupported type", struct{}{}, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, `{{ int64 .Value }}`, V(tc.in))
			if got != tc.want {
				t.Fatalf("int64 %v = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestFloat64Template exercises the type branches of the float64 conversion
// through template rendering.
func TestFloat64Template(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "0"},
		{"bool true", true, "1"},
		{"bool false", false, "0"},
		{"float32", float32(1.5), "1.5"},
		{"float64", float64(2.5), "2.5"},
		{"int", int(3), "3"},
		{"int8", int8(4), "4"},
		{"int16", int16(5), "5"},
		{"int32", int32(6), "6"},
		{"int64", int64(7), "7"},
		{"uint", uint(8), "8"},
		{"uint8", uint8(9), "9"},
		{"uint16", uint16(10), "10"},
		{"uint32", uint32(11), "11"},
		{"uint64", uint64(12), "12"},
		{"string", "13.5", "13.5"},
		{"string bad", "nope", "0"},
		{"unsupported type", struct{}{}, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, `{{ float64 .Value }}`, V(tc.in))
			if got != tc.want {
				t.Fatalf("float64 %v = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestToDecimalTemplate covers the parse-failure branch returning 0 through
// template rendering (9 is not a valid octal digit).
func TestToDecimalTemplate(t *testing.T) {
	if got := mustRender(t, `{{ todecimal .Value }}`, V("999")); got != "0" {
		t.Fatalf("todecimal 999 = %q, want 0", got)
	}
}

// TestJSONArgErrors covers the non-bool first-arg and bad-arity branches via
// template rendering.
func TestJSONArgErrors(t *testing.T) {
	if _, err := render(t, `{{ json .Value 1 }}`, V("notbool")); err == nil {
		t.Fatalf("expected error for non-bool first argument")
	}
	if _, err := render(t, `{{ json 1 2 3 }}`, nil); err == nil {
		t.Fatalf("expected error for too many arguments")
	}
	if _, err := render(t, `{{ json }}`, nil); err == nil {
		t.Fatalf("expected error for zero arguments")
	}
}

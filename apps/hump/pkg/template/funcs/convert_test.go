package funcs

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"testing"
	"unicode/utf8"
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
		{"json NaN", `{{ .Value | json }}`, math.NaN(), "null"},
		{"json nested non-finite", `{{ .Value | json }}`, map[string]any{"f": math.Inf(1), "f4": float32(math.Inf(-1)), "l": []any{1.5, math.NaN()}, "p": new(math.NaN())}, `{"f":null,"f4":null,"l":[1.5,null],"p":null}`},
		{"json pretty non-finite", `{{ .Value | json true }}`, map[string]any{"f": math.NaN()}, "{\n  \"f\": null\n}"},
		{"json bytes beside NaN", `{{ .Value | json }}`, map[string]any{"b": []byte("hi"), "f": math.NaN()}, `{"b":"aGk=","f":null}`},
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
	if _, err := render(t, `{{ json .Value }}`, V(map[string]any{"c": make(chan int), "f": math.NaN()})); err == nil {
		t.Fatalf("expected json marshal error for a channel beside NaN")
	}
	if _, err := render(t, `{{ $d := dict "f" .Value }}{{ $_ := set $d "me" $d }}{{ $d | json }}`, V(math.NaN())); err == nil {
		t.Fatalf("expected json marshal error for a cycle beside NaN")
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

func FuzzJSONRoundTrip(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add(`"quoted" \ back\slash`)
	f.Add("<script>&\u2028\u2029</script>")
	f.Add("\x00\x1f control")

	f.Fuzz(func(t *testing.T, value string) {
		if !utf8.ValidString(value) {
			t.Skip("encoding/json replaces invalid UTF-8, so only valid strings round-trip")
		}

		if got := mustRender(t, `{{ .Value | json | fromjson }}`, V(value)); got != value {
			t.Fatalf("fromjson(json(%q)) = %q", value, got)
		}
	})
}

func TestJSONNonFiniteFallback(t *testing.T) {
	got := mustRender(t, `{{ json .Value }}`, V([]any{math.NaN(), map[string]float64(nil)}))
	if got != "[null,null]" {
		t.Fatalf("json = %q, want [null,null]", got)
	}

	if _, err := render(t, `{{ json .Value }}`, V(map[int]float64{1: math.Inf(1)})); err == nil {
		t.Fatal("json of a non-string-keyed map with +Inf: expected error")
	}
}

func TestFiniteJSONValueInvalid(t *testing.T) {
	if got := finiteJSONValue(reflect.Value{}, 0); got != nil {
		t.Fatalf("finiteJSONValue(invalid) = %v, want nil", got)
	}
}

func TestInt64ClampsHugeUint(t *testing.T) {
	got := mustRender(t, `{{ int64 .Value }}`, V(uint64(math.MaxUint64)))
	if want := strconv.FormatInt(math.MaxInt64, 10); got != want {
		t.Fatalf("int64 MaxUint64 = %q, want %q", got, want)
	}
}

func TestJSONNumbersRoundTrip(t *testing.T) {
	for _, input := range []string{
		`{"big":9007199254740993,"decimal":1234567890.1234567890123456789,"nested":[-0,1e1000]}`,
		`[9007199254740993,0.000000000000000000001,1E+999]`,
		`9007199254740993`, `1.234567890123456789`, `true`, `"text"`, `null`,
	} {
		t.Run(input, func(t *testing.T) {
			if got := mustRender(t, `{{ fromjson .Value | json }}`, V(input)); got != input {
				t.Fatalf("round-trip = %s, want %s", got, input)
			}
		})
	}
	input := `[9007199254740993,{"n":1.234567890123456789}]`
	if got := mustRender(t, `{{ fromjsonarray .Value | json }}`, V(input)); got != input {
		t.Fatalf("array round-trip = %s, want %s", got, input)
	}
	if got := mustRender(t, `{{ fromjsonarray .Value | json }}`, V("null \n\t")); got != "null" {
		t.Fatalf("null array = %s", got)
	}
	if got := mustRender(t, `{{ fromjson .Value | json }}`, V(" 42 \n\t")); got != "42" {
		t.Fatalf("whitespace-wrapped number = %s", got)
	}
}

func TestJSONDecodeRejectsMalformedInput(t *testing.T) {
	for _, helper := range []string{"fromjson", "fromjsonarray"} {
		for _, input := range []string{"", " ", "{bad}", "[1,]", "[1", "[1] [2]", "null true", "[] garbage", "[] /", "01", "NaN"} {
			t.Run(helper+"/"+input, func(t *testing.T) {
				if _, err := render(t, `{{ `+helper+` .Value }}`, V(input)); err == nil {
					t.Fatalf("%s accepted %q", helper, input)
				}
			})
		}
	}
	for _, input := range []string{`{}`, `1`, `"text"`, `false`} {
		if _, err := render(t, `{{ fromjsonarray .Value }}`, V(input)); err == nil {
			t.Fatalf("fromjsonarray accepted non-array %s", input)
		}
	}
}

func TestJSONNumberConversions(t *testing.T) {
	for _, helper := range []string{"int", "int64", "float64"} {
		for _, literal := range []string{"0", "-0", "9007199254740993", "42.75", "1e3", "1e1000", "bad"} {
			t.Run(helper+"/"+literal, func(t *testing.T) {
				source := `{{ ` + helper + ` .Value }}`
				want := mustRender(t, source, V(literal))
				if got := mustRender(t, source, V(json.Number(literal))); got != want {
					t.Fatalf("number conversion = %q, string conversion = %q", got, want)
				}
			})
		}
	}
	for _, tc := range []struct{ source, want string }{
		{`{{ fromjson "9007199254740993" | int64 }}`, "9007199254740993"},
		{`{{ fromjson "2.5" | float64 }}`, "2.5"},
		{`{{ add (fromjson "2") 3 }}`, "5"},
		{`{{ mulf (fromjson "2.5") 2 }}`, "5"},
	} {
		if got := mustRender(t, tc.source, nil); got != tc.want {
			t.Fatalf("%s = %q, want %q", tc.source, got, tc.want)
		}
	}
}

func TestJSONNumberComparisonAndTruthiness(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{`{{ $n := fromjson "0" }}{{ if $n }}truthy{{ else }}false{{ end }}`, "truthy"},
		{`{{ empty (fromjson "0") }}`, "false"},
		{`{{ eq (fromjson "0") "0" }}`, "true"},
		{`{{ eq (fromjson "0") (fromjson "0.0") }}`, "false"},
		{`{{ lt (fromjson "10") (fromjson "2") }}`, "true"},
		{`{{ fromjson "0" | int64 | empty }}`, "true"},
	} {
		if got := mustRender(t, tc.source, nil); got != tc.want {
			t.Fatalf("%s = %q, want %q", tc.source, got, tc.want)
		}
	}
	if _, err := render(t, `{{ eq (fromjson "0") 0 }}`, nil); err == nil {
		t.Fatal("json.Number must require explicit conversion before numeric comparison")
	}
}

func TestJSONNonFiniteFallbackPreservesNumbers(t *testing.T) {
	value := map[string]any{
		"bad":    math.Inf(1),
		"nested": []any{json.Number("9007199254740993"), map[string]any{"n": json.Number("1.234567890123456789"), "nan": math.NaN()}},
	}
	want := `{"bad":null,"nested":[9007199254740993,{"n":1.234567890123456789,"nan":null}]}`
	if got := mustRender(t, `{{ .Value | json }}`, V(value)); got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

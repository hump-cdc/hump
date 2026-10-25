package funcs

import (
	"strings"
	"testing"
)

const (
	validTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	validTraceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
	validSpanID      = "00f067aa0ba902b7"
)

func TestTraceIDTemplate(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"valid", validTraceparent, validTraceID},
		{"empty", "", ""},
		{"too_few_parts", "00-" + validTraceID + "-" + validSpanID, "00-" + validTraceID + "-" + validSpanID},
		{"too_many_parts", validTraceparent + "-extra", validTraceparent + "-extra"},
		{"version_ff", "ff-" + validTraceID + "-" + validSpanID + "-01", "ff-" + validTraceID + "-" + validSpanID + "-01"},
		{"version_short", "0-" + validTraceID + "-" + validSpanID + "-01", "0-" + validTraceID + "-" + validSpanID + "-01"},
		{"version_uppercase", "AA-" + validTraceID + "-" + validSpanID + "-01", "AA-" + validTraceID + "-" + validSpanID + "-01"},
		{"trace_uppercase", "00-4BF92F3577B34DA6A3CE929D0E0E4736-" + validSpanID + "-01", "00-4BF92F3577B34DA6A3CE929D0E0E4736-" + validSpanID + "-01"},
		{"trace_all_zero", "00-00000000000000000000000000000000-" + validSpanID + "-01", "00-00000000000000000000000000000000-" + validSpanID + "-01"},
		{"trace_short", "00-4bf92f3577b34da6a3ce929d0e0e473-" + validSpanID + "-01", "00-4bf92f3577b34da6a3ce929d0e0e473-" + validSpanID + "-01"},
		{"trace_non_hex", "00-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz-" + validSpanID + "-01", "00-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz-" + validSpanID + "-01"},
		{"span_all_zero", "00-" + validTraceID + "-0000000000000000-01", "00-" + validTraceID + "-0000000000000000-01"},
		{"span_short", "00-" + validTraceID + "-00f067aa0ba902b-01", "00-" + validTraceID + "-00f067aa0ba902b-01"},
		{"flags_short", "00-" + validTraceID + "-" + validSpanID + "-1", "00-" + validTraceID + "-" + validSpanID + "-1"},
		{"flags_non_hex", "00-" + validTraceID + "-" + validSpanID + "-zz", "00-" + validTraceID + "-" + validSpanID + "-zz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, `{{ trace_id .Value }}`, V(tc.input))
			if got != tc.want {
				t.Fatalf("trace_id(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSpanIDTemplate(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"valid", validTraceparent, validSpanID},
		{"empty", "", ""},
		{"too_few_parts", "00-" + validTraceID + "-" + validSpanID, ""},
		{"too_many_parts", validTraceparent + "-extra", ""},
		{"version_ff", "ff-" + validTraceID + "-" + validSpanID + "-01", ""},
		{"trace_all_zero", "00-00000000000000000000000000000000-" + validSpanID + "-01", ""},
		{"span_all_zero", "00-" + validTraceID + "-0000000000000000-01", ""},
		{"span_short", "00-" + validTraceID + "-00f067aa0ba902b-01", ""},
		{"span_uppercase", "00-" + validTraceID + "-00F067AA0BA902B7-01", ""},
		{"flags_non_hex", "00-" + validTraceID + "-" + validSpanID + "-zz", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, `{{ span_id .Value }}`, V(tc.input))
			if got != tc.want {
				t.Fatalf("span_id(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func FuzzTraceparent(f *testing.F) {
	f.Add(validTraceparent)
	f.Add("ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	f.Add("00-00000000000000000000000000000000-00f067aa0ba902b7-01")
	f.Add("00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01")
	f.Add("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra")
	f.Add("")

	f.Fuzz(func(t *testing.T, traceparent string) {
		traceID := mustRender(t, `{{ .Value | trace_id }}`, V(traceparent))
		spanID := mustRender(t, `{{ .Value | span_id }}`, V(traceparent))

		if spanID == "" {
			if traceID != traceparent {
				t.Fatalf("trace_id(%q) = %q, want the input back when span_id rejects it", traceparent, traceID)
			}

			return
		}

		parts := strings.Split(traceparent, "-")
		if len(parts) != 4 || parts[1] != traceID || parts[2] != spanID {
			t.Fatalf("trace_id, span_id of %q = %q, %q, want its second and third fields", traceparent, traceID, spanID)
		}

		for _, id := range []string{traceID, spanID} {
			if strings.Trim(id, "0123456789abcdef") != "" || strings.Trim(id, "0") == "" {
				t.Fatalf("id %q of %q is not non-zero lowercase hex", id, traceparent)
			}
		}

		if len(traceID) != 32 || len(spanID) != 16 {
			t.Fatalf("trace_id, span_id of %q = %q, %q, want 32 and 16 hex digits", traceparent, traceID, spanID)
		}
	})
}

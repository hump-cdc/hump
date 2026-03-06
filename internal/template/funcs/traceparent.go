package funcs

import "strings"

// W3C Trace Context (Level 1) traceparent format:
//
//	version "-" trace-id "-" parent-id "-" trace-flags
//	   2          32            16             2          (lowercase hex chars)
//
// Spec also marks version "ff" and all-zero trace-id / parent-id as invalid.
// https://www.w3.org/TR/trace-context/

const (
	traceparentVersionLen  = 2
	traceparentTraceIDLen  = 32
	traceparentParentIDLen = 16
	traceparentFlagsLen    = 2
)

// tmplTraceID extracts the trace-id from a W3C traceparent string. Returns
// the source string unchanged when the input fails strict format validation.
func tmplTraceID(traceparent string) string {
	version, traceID, parentID, flags, ok := splitTraceparent(traceparent)
	if !ok {
		return traceparent
	}
	if !isValidTraceparent(version, traceID, parentID, flags) {
		return traceparent
	}
	return traceID
}

// tmplSpanID extracts the parent-id (a.k.a. span-id) from a W3C traceparent
// string. Returns "" when the input fails strict format validation.
func tmplSpanID(traceparent string) string {
	version, traceID, parentID, flags, ok := splitTraceparent(traceparent)
	if !ok {
		return ""
	}
	if !isValidTraceparent(version, traceID, parentID, flags) {
		return ""
	}
	return parentID
}

func splitTraceparent(s string) (version, traceID, parentID, flags string, ok bool) {
	parts := strings.Split(s, "-")
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}

func isValidTraceparent(version, traceID, parentID, flags string) bool {
	if len(version) != traceparentVersionLen || !isLowerHex(version) || version == "ff" {
		return false
	}
	if len(traceID) != traceparentTraceIDLen || !isLowerHex(traceID) || isAllZero(traceID) {
		return false
	}
	if len(parentID) != traceparentParentIDLen || !isLowerHex(parentID) || isAllZero(parentID) {
		return false
	}
	if len(flags) != traceparentFlagsLen || !isLowerHex(flags) {
		return false
	}
	return true
}

func isLowerHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

func isAllZero(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return false
		}
	}
	return true
}

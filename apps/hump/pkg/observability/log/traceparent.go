package log

const (
	traceparentLen = 55
	traceIDStart   = 3
	traceIDEnd     = 35
	spanIDStart    = 36
	spanIDEnd      = 52
	versionEnd     = 2
	flagsStart     = 53
)

// parseTraceparent parses a W3C traceparent header.
// Expected format: "vv-<32 hex trace_id>-<16 hex span_id>-ff" (55 chars).
// Returns sub-slices of the input string, so it does not allocate.
func parseTraceparent(traceparent string) (traceID, spanID string, ok bool) {
	if len(traceparent) != traceparentLen {
		return "", "", false
	}
	if traceparent[versionEnd] != '-' || traceparent[traceIDEnd] != '-' || traceparent[spanIDEnd] != '-' {
		return "", "", false
	}
	if !isHex(traceparent[:versionEnd]) ||
		!isHex(traceparent[traceIDStart:traceIDEnd]) ||
		!isHex(traceparent[spanIDStart:spanIDEnd]) ||
		!isHex(traceparent[flagsStart:]) {
		return "", "", false
	}
	return traceparent[traceIDStart:traceIDEnd], traceparent[spanIDStart:spanIDEnd], true
}

func isHex(s string) bool {
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

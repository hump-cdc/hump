// Package log configures the slog default logger from the environment and
// derives scoped loggers from it.
package log

import (
	"hump/pkg/version"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"golang.org/x/term"
)

// Log is a slog logger carrying the build and scope attributes.
type Log struct {
	*slog.Logger
}

// Initializes slog default logger.
//
// The default handler is selected via environment variables:
//   - LOG_FORMAT=json selects [slog.NewJSONHandler]; otherwise a colored
//     [tint.NewHandler] is used (color is disabled when stdout is not a TTY).
//   - LOG_VERBOSE=true raises the level from Info to Debug.
func init() {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_VERBOSE"), "true") {
		level = slog.LevelDebug
	}

	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "json") {
		slog.SetDefault(
			slog.New(
				slog.NewJSONHandler(os.Stdout,
					&slog.HandlerOptions{
						Level:     level,
						AddSource: false,
					})))
	} else {
		slog.SetDefault(
			slog.New(
				tint.NewTextHandler(os.Stdout,
					&tint.Options{
						TimeFormat: time.Kitchen,
						Level:      level,
						NoColor:    !term.IsTerminal(int(os.Stdout.Fd())),
						AddSource:  false,
					})))
	}
}

// Scope returns a [Log] derived from [slog.Default], pre-populated with the
// build's commit and version attributes (when set) and, if area is non-empty,
// an "area" attribute identifying the calling subsystem.
func Scope(area string) *Log {
	logger := slog.Default()
	if version.Commit != "" {
		logger = logger.With("commit", version.Commit)
	}
	if version.Version != "" {
		logger = logger.With("version", version.Version)
	}

	if area != "" {
		return &Log{Logger: logger.With("area", area)}
	}

	return &Log{Logger: logger}
}

// WithScope returns a child [Log] with the "area" attribute set to area,
// overriding any previously set scope.
func (l *Log) WithScope(area string) *Log {
	return &Log{Logger: l.With("area", area)}
}

// WithTraceparent returns a child [Log] enriched with trace correlation
// attributes parsed from a W3C traceparent header value.
//
// If traceparent is empty, l is returned unchanged. If it parses successfully,
// the returned logger carries "trace_id" and "span_id" attributes; otherwise
// the raw value is attached as "trace_id" as a best-effort fallback.
func (l *Log) WithTraceparent(traceparent string) *Log {
	if traceparent == "" {
		return l
	}

	if traceID, spanID, ok := parseTraceparent(traceparent); ok {
		return &Log{Logger: l.With("trace_id", traceID, "span_id", spanID)}
	}

	return &Log{Logger: l.With("trace_id", traceparent)}
}

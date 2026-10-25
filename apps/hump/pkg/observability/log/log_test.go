package log

import (
	"bytes"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"hump/pkg/version"
)

func TestInitHonoursEnvironment(t *testing.T) {
	if os.Getenv("HUMP_LOG_INIT_CHILD") != "" {
		slog.Debug("probe")
		return
	}

	// init reads the environment once per process, so the JSON, Debug-level
	// branch runs in a child test binary started with it set.
	args := []string{"-test.run=^TestInitHonoursEnvironment$"}
	if dir := flag.Lookup("test.gocoverdir"); dir != nil && dir.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+dir.Value.String())
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "HUMP_LOG_INIT_CHILD=1", "LOG_FORMAT=json", "LOG_VERBOSE=true")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("child test binary: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), `"level":"DEBUG","msg":"probe"`) {
		t.Fatalf("child output lacks a JSON debug line:\n%s", out)
	}
}

func TestScope(t *testing.T) {
	tests := map[string]struct {
		area    string
		commit  string
		version string
		want    map[string]string
		absent  []string
	}{
		"area and build": {
			area: "wal", commit: "abc123", version: "1.2.3",
			want: map[string]string{"area": "wal", "commit": "abc123", "version": "1.2.3"},
		},
		"no area, no build": {
			absent: []string{"area", "commit", "version"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			setBuild(t, tt.commit, tt.version)
			buf := captureDefault(t)

			Scope(tt.area).Info("hello")

			record := decode(t, buf)
			for k, v := range tt.want {
				if record[k] != v {
					t.Errorf("%s = %v, want %q", k, record[k], v)
				}
			}
			for _, k := range tt.absent {
				if _, ok := record[k]; ok {
					t.Errorf("%s = %v, want absent", k, record[k])
				}
			}
		})
	}
}

func TestWithScope(t *testing.T) {
	buf := captureDefault(t)

	Scope("").WithScope("sink").Info("hello")

	if got := decode(t, buf)["area"]; got != "sink" {
		t.Errorf("area = %v, want sink", got)
	}
}

func TestWithTraceparent(t *testing.T) {
	const valid = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	tests := map[string]struct {
		traceparent string
		wantTrace   any
		wantSpan    any
	}{
		"valid": {
			traceparent: valid,
			wantTrace:   "4bf92f3577b34da6a3ce929d0e0e4736",
			wantSpan:    "00f067aa0ba902b7",
		},
		"uppercase hex": {
			traceparent: "00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
			wantTrace:   "4BF92F3577B34DA6A3CE929D0E0E4736",
			wantSpan:    "00F067AA0BA902B7",
		},
		"empty": {},
		"wrong length": {
			traceparent: "00-abc",
			wantTrace:   "00-abc",
		},
		"misplaced dash": {
			traceparent: "00_4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantTrace:   "00_4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		},
		"non-hex digit": {
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e473z-00f067aa0ba902b7-01",
			wantTrace:   "00-4bf92f3577b34da6a3ce929d0e0e473z-00f067aa0ba902b7-01",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			buf := captureDefault(t)

			Scope("").WithTraceparent(tt.traceparent).Info("hello")

			record := decode(t, buf)
			if record["trace_id"] != tt.wantTrace {
				t.Errorf("trace_id = %v, want %v", record["trace_id"], tt.wantTrace)
			}
			if record["span_id"] != tt.wantSpan {
				t.Errorf("span_id = %v, want %v", record["span_id"], tt.wantSpan)
			}
		})
	}
}

// setBuild sets the build metadata for the test and restores it afterwards.
func setBuild(t *testing.T, commit, ver string) {
	t.Helper()

	prevCommit, prevVersion := version.Commit, version.Version
	version.Commit, version.Version = commit, ver
	t.Cleanup(func() { version.Commit, version.Version = prevCommit, prevVersion })
}

// captureDefault points slog.Default at a JSON handler writing to the returned
// buffer, restoring the previous default afterwards.
func captureDefault(t *testing.T) *bytes.Buffer {
	t.Helper()

	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	return &buf
}

// decode parses the single JSON log record in buf.
func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}

	return record
}

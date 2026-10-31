package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	cfg "hump/config"
	sinkcfg "hump/config/sink"
	"hump/pkg/observability/log"
)

func TestConfigurationLogsOnlyTableAndSinkTypes(t *testing.T) {
	if os.Getenv("HUMP_CONFIG_LOG_CHILD") == "1" {
		configuration, err := cfg.Parse(`tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: https://sentinel-user:sentinel-password@example.com/sentinel-path?token=sentinel-token
          headers:
            Authorization: Bearer sentinel-header
          body: 'sentinel-body {{ .Next.secret }}'
      - http:
          url: https://example.com/other
`)
		if err != nil {
			t.Fatal(err)
		}
		logConfiguration(configuration)
		return
	}

	for _, verbose := range []string{"false", "true"} {
		t.Run("verbose="+verbose, func(t *testing.T) {
			cmd := configurationLogCommand(t)
			cmd.Env = append(os.Environ(), "HUMP_CONFIG_LOG_CHILD=1", "LOG_FORMAT=json", "LOG_VERBOSE="+verbose)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("child: %v\n%s", err, out)
			}
			for _, forbidden := range []string{"sentinel-", "example.com", ".Next.secret", "Authorization", "https://"} {
				if strings.Contains(string(out), forbidden) {
					t.Errorf("logs contain %q: %s", forbidden, out)
				}
			}
			count := 0
			for _, line := range strings.Split(string(out), "\n") {
				if !strings.HasPrefix(line, "{") {
					continue
				}
				count++
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["schema"] != "public" || record["table"] != "users" || record["sink_count"] != float64(2) {
					t.Errorf("unexpected summary: %s", line)
				}
				types, ok := record["sink_types"].(map[string]any)
				if !ok || len(types) != 1 || types["http"] != float64(2) {
					t.Errorf("unexpected sink types: %s", line)
				}
			}
			want := 0
			if verbose == "true" {
				want = 1
			}
			if count != want {
				t.Errorf("summary count = %d, want %d: %s", count, want, out)
			}
		})
	}
}

func TestRunLogsSafeConfigurationBeforeDatabaseFailure(t *testing.T) {
	if os.Getenv("HUMP_CONFIG_RUN_CHILD") == "1" {
		os.Args = []string{os.Args[0], "--database-url=postgres://localhost:not-a-port/hump", "--config", `tables:
  - schema: public
    table: users
    sinks:
      - http:
          url: https://startup-secret:startup-password@example.com/private?token=startup-token
          headers:
            Authorization: Bearer startup-header
          body: startup-payload
`}
		for _, key := range []string{"WEB_PORT", "TELEMETRY_PORT"} {
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
		if code := run(); code != 1 {
			t.Fatalf("run() = %d, want 1 for invalid database port", code)
		}
		return
	}

	cmd := configurationLogCommand(t)
	cmd.Env = append(os.Environ(), "HUMP_CONFIG_RUN_CHILD=1", "LOG_FORMAT=json", "LOG_VERBOSE=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child: %v\n%s", err, out)
	}
	for _, secret := range []string{"startup-", "example.com", "Authorization", "https://"} {
		if bytes.Contains(out, []byte(secret)) {
			t.Errorf("startup logs contain %q: %s", secret, out)
		}
	}
	var summaries int
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record["msg"] == "Configuration" {
			summaries++
			if record["schema"] != "public" || record["table"] != "users" || record["sink_count"] != float64(1) {
				t.Errorf("unexpected startup summary: %s", line)
			}
		}
	}
	if summaries != 1 {
		t.Fatalf("startup summaries = %d, want 1: %s", summaries, out)
	}
}

func TestLogConfigurationCountsSinkTypesPerTable(t *testing.T) {
	var output bytes.Buffer
	previous := logger
	logger = &log.Log{Logger: slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	t.Cleanup(func() { logger = previous })
	configuration := cfg.Configuration{Tables: []cfg.Table{
		{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
			{Sink: &sinkcfg.HTTPSink{}}, {Sink: &sinkcfg.HTTPSink{}}, {Sink: &sinkcfg.KafkaSink{}}, {},
		}},
		{Schema: "audit", Name: "users", Sinks: []sinkcfg.Config{{}}},
		{Schema: "public", Name: "empty"},
	}}

	logConfiguration(cfg.Configuration{})
	if output.Len() != 0 {
		t.Fatalf("empty configuration logged: %s", output.String())
	}
	logConfiguration(configuration)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != len(configuration.Tables) {
		t.Fatalf("summary count = %d, want %d: %s", len(lines), len(configuration.Tables), output.String())
	}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		table := configuration.Tables[i]
		if record["level"] != "DEBUG" || record["msg"] != "Configuration" || record["schema"] != string(table.Schema) || record["table"] != string(table.Name) || record["sink_count"] != float64(len(table.Sinks)) {
			t.Errorf("unexpected summary: %s", line)
		}
		types, ok := record["sink_types"].(map[string]any)
		if !ok {
			t.Fatalf("sink_types is not an object: %s", line)
		}
		if i == 0 {
			if len(types) != 2 || types["http"] != float64(2) || types["kafka"] != float64(1) {
				t.Errorf("unexpected sink types: %s", line)
			}
		} else if len(types) != 0 {
			t.Errorf("sink counts carried over between tables: %s", line)
		}
		if len(record) != 7 {
			t.Errorf("summary has unexpected fields: %s", line)
		}
	}
}

// configurationLogCommand preserves child coverage and bounds startup failures.
func configurationLogCommand(t *testing.T) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	args := []string{"-test.run=^" + t.Name() + "$"}
	if parent, _, ok := strings.Cut(t.Name(), "/"); ok {
		args[0] = "-test.run=^" + parent + "$"
	}
	if dir := flag.Lookup("test.gocoverdir"); dir != nil && dir.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+dir.Value.String())
	}
	return exec.CommandContext(ctx, os.Args[0], args...)
}

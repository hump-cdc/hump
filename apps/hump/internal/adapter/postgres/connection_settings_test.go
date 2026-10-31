package postgres

import (
	"bytes"
	"context"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDefaultSettingsConfigRemovesOnlyTargetOverrides(t *testing.T) {
	t.Setenv("PGOPTIONS", "-c standard_conforming_strings=off -c statement_timeout=1000")
	for _, options := range []string{
		"-c standard_conforming_strings=off -c statement_timeout=1000",
		"-cstandard_conforming_strings=off -c statement_timeout=1000",
		"--standard-conforming-strings=off -c statement_timeout=1000",
		`-c standard\_conforming_strings=off -c statement_timeout=1000`,
	} {
		dsn := "postgres://user:secret@localhost/db?application_name=keep&standard_conforming_strings=off&options=" + strings.ReplaceAll(url.QueryEscape(options), "+", "%20")
		cfg, err := defaultSettingsConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := cfg.RuntimeParams["standard_conforming_strings"]; ok {
			t.Fatal("diagnostic contains client override")
		}
		if cfg.RuntimeParams["options"] != "-c statement_timeout=1000" || cfg.RuntimeParams["application_name"] != "keep" || cfg.Password != "secret" {
			t.Fatalf("unrelated connection settings not preserved: options=%q application=%q", cfg.RuntimeParams["options"], cfg.RuntimeParams["application_name"])
		}
	}
	cfg, err := defaultSettingsConfig("postgres://user@localhost/db")
	if err != nil || cfg.RuntimeParams["options"] != "-c statement_timeout=1000" {
		t.Fatalf("PGOPTIONS not sanitized: %v", err)
	}
}

func TestBackendOptionsPreserveEscapingAndRepeatedSettings(t *testing.T) {
	options := `-c application_name=hello\ world --standard_conforming_strings=on -cstandard_conforming_strings=off -c search_path=public`
	got, err := withoutStringSettingOptions(options)
	if err != nil || got != `-c application_name=hello\ world -c search_path=public` {
		t.Fatalf("cleaned options = %q, %v", got, err)
	}
	if _, err := withoutStringSettingOptions(`-c application_name=bad\`); err == nil {
		t.Fatal("expected malformed option error")
	}
}

func TestDefaultSettingDiagnosticsAreNonblockingAndSafe(t *testing.T) {
	for _, state := range []string{"on", "off", "secret-sentinel", "error", "empty", "invalid-dsn"} {
		t.Run(state, func(t *testing.T) {
			var logs bytes.Buffer
			captureSettingsLogs(t, &logs)
			server := newFakeServer(t)
			server.setResponder(func(string) queryResult {
				if state == "error" {
					return serverError("42501", "secret-sentinel")
				}
				if state == "empty" {
					return queryResult{}
				}
				return queryResult{columns: 1, rows: [][][]byte{{[]byte(state)}}}
			})
			dsn := fakeServerUrl(server) + "&standard_conforming_strings=on&options=" + strings.ReplaceAll(url.QueryEscape("-c standard_conforming_strings=on"), "+", "%20")
			if state == "invalid-dsn" {
				dsn = "postgres://secret-sentinel:secret-sentinel@invalid host/db"
			}
			checkStandardConformingStringsDefault(context.Background(), dsn)
			if strings.Contains(logs.String(), "secret-sentinel") {
				t.Fatalf("diagnostic leaked secret: %s", logs.String())
			}
			if got := strings.Contains(logs.String(), "WARN"); got != (state != "on") {
				t.Fatalf("unexpected diagnostic logs: %s", logs.String())
			}
			if state != "invalid-dsn" {
				server.mu.Lock()
				if len(server.startupParams) == 0 {
					server.mu.Unlock()
					t.Fatal("diagnostic did not connect")
				}
				params := server.startupParams[0]
				server.mu.Unlock()
				if _, ok := params["standard_conforming_strings"]; ok || params["options"] != "" {
					t.Fatalf("diagnostic startup overrides effective default: %v", params)
				}
				deadline := time.Now().Add(time.Second)
				for {
					server.mu.Lock()
					closed := server.closedSessions > 0
					server.mu.Unlock()
					if closed {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("diagnostic session not closed")
					}
					time.Sleep(time.Millisecond)
				}
			}
		})
	}
}

func TestSessionStringSettingsWarnWithoutRejectingOrRepeating(t *testing.T) {
	for _, state := range []string{"on", "off", "", "secret-sentinel"} {
		for _, replication := range []bool{false, true} {
			t.Run(state+map[bool]string{false: "/ordinary", true: "/replication"}[replication], func(t *testing.T) {
				var logs bytes.Buffer
				captureSettingsLogs(t, &logs)
				server := newFakeServer(t)
				server.mu.Lock()
				server.stringSetting = &state
				server.mu.Unlock()
				opts := []option{withMaxAttempts(1)}
				if replication {
					opts = append(opts, withReplication())
				}
				conn, err := newConnection(fakeServerUrl(server)+"&standard_conforming_strings=off", opts...)
				if err != nil {
					t.Fatal(err)
				}
				defer closeDetached(conn.Close)
				for range 2 {
					if _, err := conn.Exec(context.Background(), "SELECT 1"); err != nil {
						t.Fatal(err)
					}
				}
				want := 0
				if state != "on" {
					want = 1
				}
				if strings.Count(logs.String(), "WARN") != want || strings.Contains(logs.String(), "secret-sentinel") {
					t.Fatalf("unexpected session logs: %s", logs.String())
				}
				if err := conn.Close(context.Background()); err != nil {
					t.Fatal(err)
				}
				if _, err := conn.Exec(context.Background(), "SELECT 1"); err != nil {
					t.Fatal(err)
				}
				if strings.Count(logs.String(), "WARN") != 2*want {
					t.Fatalf("reconnect warning missing: %s", logs.String())
				}
				server.mu.Lock()
				defer server.mu.Unlock()
				for _, params := range server.startupParams {
					if params["standard_conforming_strings"] != "on" || (replication && params["replication"] != "database") {
						t.Fatalf("incorrect startup parameters")
					}
				}
			})
		}
	}
}

func TestSessionSettingChangeOnReacquisitionWarns(t *testing.T) {
	var logs bytes.Buffer
	captureSettingsLogs(t, &logs)
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult {
		return queryResult{stringSetting: "off"}
	})
	conn, err := newConnection(fakeServerUrl(server))
	if err != nil {
		t.Fatal(err)
	}
	defer closeDetached(conn.Close)
	_, err = conn.Raw(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatalf("normal session warned: %s", logs.String())
	}
	if _, err := conn.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Raw(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "WARN") {
		t.Fatal("changed runtime setting was not diagnosed on reacquisition")
	}
}

// captureSettingsLogs installs a JSON logger for non-parallel diagnostic tests.
func captureSettingsLogs(t *testing.T, output *bytes.Buffer) {
	t.Helper()
	previous := logger.Logger
	logger.Logger = slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	t.Cleanup(func() { logger.Logger = previous })
}

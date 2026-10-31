package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	cfg "hump/config"
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
			cmd := exec.Command(os.Args[0], "-test.run=^TestConfigurationLogsOnlyTableAndSinkTypes$")
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

package options

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestParseReadsConfigFlag(t *testing.T) {
	setup(t, "-c", "tables: []", "-d", "postgres://db")

	o, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if o.Config != "tables: []" {
		t.Errorf("Config = %q, want %q", o.Config, "tables: []")
	}

	if o.DatabaseUrl != "postgres://db" {
		t.Errorf("DatabaseUrl = %q, want %q", o.DatabaseUrl, "postgres://db")
	}

	if o.PublicationName != "hump_pub" || o.ReplicationSlotName != "hump_slot" {
		t.Errorf("defaults = %q, %q, want hump_pub, hump_slot", o.PublicationName, o.ReplicationSlotName)
	}
}

func TestParseReadsConfigFileFlag(t *testing.T) {
	path := writeConfig(t, "from file")
	setup(t, "-f", path)

	o, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if o.Config != "from file" {
		t.Errorf("Config = %q, want %q", o.Config, "from file")
	}
}

func TestParseRejectsConfigConflict(t *testing.T) {
	setup(t, "-c", "inline", "-f", "config.yaml")

	if _, err := Parse(); !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("Parse() error = %v, want ErrConfigConflict", err)
	}
}

func TestParseReturnsConfigFileError(t *testing.T) {
	setup(t, "-f", filepath.Join(t.TempDir(), "missing.yaml"))

	if _, err := Parse(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Parse() error = %v, want fs.ErrNotExist", err)
	}
}

func TestParseReadsConfigEnv(t *testing.T) {
	setup(t)
	t.Setenv("CONFIG", "from env")

	o, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if o.Config != "from env" {
		t.Errorf("Config = %q, want %q", o.Config, "from env")
	}
}

func TestParseReadsConfigFileEnv(t *testing.T) {
	setup(t)
	t.Setenv("CONFIG_FILE", writeConfig(t, "from env file"))

	o, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if o.Config != "from env file" {
		t.Errorf("Config = %q, want %q", o.Config, "from env file")
	}
}

func TestParseReadsConfigFromStdin(t *testing.T) {
	setup(t)
	setStdin(t, writeConfig(t, "from stdin"))

	o, err := Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if o.Config != "from stdin" {
		t.Errorf("Config = %q, want %q", o.Config, "from stdin")
	}
}

func TestParseReturnsStdinReadError(t *testing.T) {
	setup(t)
	// A directory stats as a non-character device but fails to read.
	setStdin(t, t.TempDir())

	if _, err := Parse(); err == nil {
		t.Fatal("Parse() error = nil, want a read error")
	}
}

func TestParseRequiresConfig(t *testing.T) {
	setup(t)

	if _, err := Parse(); !errors.Is(err, ErrConfigMissing) {
		t.Fatalf("Parse() error = %v, want ErrConfigMissing", err)
	}
}

func TestParseReturnsErrHelp(t *testing.T) {
	setup(t, "--help")

	if _, err := Parse(); !errors.Is(err, ErrHelp) {
		t.Fatalf("Parse() error = %v, want ErrHelp", err)
	}
}

func TestParseReturnsFlagError(t *testing.T) {
	setup(t, "--unknown")

	_, err := Parse()
	if err == nil || errors.Is(err, ErrHelp) {
		t.Fatalf("Parse() error = %v, want a flag error", err)
	}
}

// setup points Parse at args, clears the config env vars and sets stdin to
// /dev/null, a character device that Parse does not read config from.
func setup(t *testing.T, args ...string) {
	t.Helper()

	oldArgs := os.Args
	os.Args = append([]string{"hump"}, args...)
	t.Cleanup(func() { os.Args = oldArgs })

	t.Setenv("CONFIG", "")
	t.Setenv("CONFIG_FILE", "")

	setStdin(t, os.DevNull)
}

// setStdin replaces os.Stdin with the file at path for the test's duration.
func setStdin(t *testing.T, path string) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = f.Close()
	})
}

// writeConfig writes content to a temp file and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}

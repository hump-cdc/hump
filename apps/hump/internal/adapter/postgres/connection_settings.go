package postgres

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// standardConformingStringsSQL reads only the effective setting of this role/database.
//
//go:embed sql/standard_conforming_strings.sql
var standardConformingStringsSQL string

// checkStandardConformingStringsDefault diagnoses defaults without blocking bootstrap on failure.
func checkStandardConformingStringsDefault(ctx context.Context, connectionString string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cfg, err := defaultSettingsConfig(connectionString)
	if err != nil {
		logger.Warn("Unable to diagnose PostgreSQL standard_conforming_strings default")
		return
	}
	conn, err := pgconn.ConnectConfig(ctx, cfg)
	if err != nil {
		logger.Warn("Unable to diagnose PostgreSQL standard_conforming_strings default")
		return
	}
	defer closeDetached(conn.Close)

	results, err := conn.Exec(ctx, standardConformingStringsSQL).ReadAll()
	if err != nil || len(results) != 1 || len(results[0].Rows) != 1 || len(results[0].Rows[0]) != 1 {
		logger.Warn("Unable to diagnose PostgreSQL standard_conforming_strings default")
		return
	}
	state := stringSettingState(string(results[0].Rows[0][0]))
	if state != "on" {
		logger.Warn("PostgreSQL default standard_conforming_strings is not on", slog.String("state", state))
	}
}

// defaultSettingsConfig removes only client overrides of the diagnosed setting.
func defaultSettingsConfig(connectionString string) (*pgconn.Config, error) {
	cfg, err := pgconn.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}
	for name := range cfg.RuntimeParams {
		if isStringSetting(name) {
			delete(cfg.RuntimeParams, name)
		}
	}
	if options, ok := cfg.RuntimeParams["options"]; ok {
		cleaned, err := withoutStringSettingOptions(options)
		if err != nil {
			return nil, err
		}
		if cleaned == "" {
			delete(cfg.RuntimeParams, "options")
		} else {
			cfg.RuntimeParams["options"] = cleaned
		}
	}
	return cfg, nil
}

// stringSettingState restricts logged server data to known, non-sensitive states.
func stringSettingState(value string) string {
	if value == "on" || value == "off" {
		return value
	}
	return "unknown"
}

// isStringSetting recognizes GUC names, including the long-option hyphen spelling.
func isStringSetting(name string) bool {
	return strings.EqualFold(strings.ReplaceAll(name, "-", "_"), "standard_conforming_strings")
}

// withoutStringSettingOptions preserves other backend options and their escaping.
func withoutStringSettingOptions(options string) (string, error) {
	tokens, err := splitBackendOptions(options)
	if err != nil {
		return "", err
	}
	kept := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		assignment := ""
		pair := token.value == "-c"
		switch {
		case pair && i+1 < len(tokens):
			assignment = tokens[i+1].value
		case strings.HasPrefix(token.value, "-c"):
			assignment = strings.TrimPrefix(token.value, "-c")
		case strings.HasPrefix(token.value, "--"):
			assignment = strings.TrimPrefix(token.value, "--")
		}
		name, _, _ := strings.Cut(assignment, "=")
		if isStringSetting(name) {
			if pair {
				i++
			}
			continue
		}
		kept = append(kept, token.raw)
	}
	return strings.Join(kept, " "), nil
}

// backendOption retains the wire spelling while exposing the backend-decoded argument.
type backendOption struct {
	raw   string
	value string
}

// splitBackendOptions matches PostgreSQL's whitespace and backslash option splitting, not shell quoting.
func splitBackendOptions(options string) ([]backendOption, error) {
	var tokens []backendOption
	for i := 0; i < len(options); {
		if strings.ContainsRune(" \t\n\r\v\f", rune(options[i])) {
			i++
			continue
		}
		start := i
		var value strings.Builder
		for i < len(options) && !strings.ContainsRune(" \t\n\r\v\f", rune(options[i])) {
			if options[i] == '\\' {
				i++
				if i == len(options) {
					return nil, errors.New("incomplete backend option escape")
				}
			}
			value.WriteByte(options[i])
			i++
		}
		tokens = append(tokens, backendOption{raw: options[start:i], value: value.String()})
	}
	return tokens, nil
}

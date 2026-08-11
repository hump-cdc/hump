package catalog

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"

	"hump/internal/adapter/postgres/connection"
	"hump/internal/types/errs"
)

//go:embed sql/config.sql
var configSql string

// WalLevel mirrors the PostgreSQL wal_level GUC.
type WalLevel string

// PostgreSQL wal_level values. Logical decoding requires WalLevelLogical.
const (
	WalLevelMinimal WalLevel = "minimal"
	WalLevelReplica WalLevel = "replica"
	WalLevelLogical WalLevel = "logical"
)

// Config captures the subset of server settings relevant to logical
// replication setup. Version is the major version (e.g. 15, 16).
type Config struct {
	Version             int
	WalLevel            WalLevel
	MaxReplicationSlots int
	MaxWalSenders       int
	HasTimescaleDB      bool
}

// GetConfig queries the server for version, WAL settings, and the presence of
// the TimescaleDB extension. Returns errs.ErrDbSetup-wrapped errors on missing
// or unparseable rows.
func GetConfig(ctx context.Context, conn *connection.Connection) (*Config, error) {
	results, err := conn.Exec(ctx, configSql)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || len(results[0].Rows) == 0 {
		return nil, fmt.Errorf("empty database config result: %w", errs.ErrDbSetup)
	}

	row := results[0].Rows[0]

	version, err := strconv.Atoi(string(row[0]))
	if err != nil {
		return nil, fmt.Errorf("parse server_version_num: %w", errors.Join(err, errs.ErrDbSetup))
	}
	maxReplicationSlots, err := strconv.Atoi(string(row[2]))
	if err != nil {
		return nil, fmt.Errorf("parse max_replication_slots: %w", errors.Join(err, errs.ErrDbSetup))
	}
	maxWalSenders, err := strconv.Atoi(string(row[3]))
	if err != nil {
		return nil, fmt.Errorf("parse max_wal_senders: %w", errors.Join(err, errs.ErrDbSetup))
	}

	return &Config{
		Version:             version / 10000,
		WalLevel:            WalLevel(row[1]),
		MaxReplicationSlots: maxReplicationSlots,
		MaxWalSenders:       maxWalSenders,
		HasTimescaleDB:      string(row[4]) == "t",
	}, nil
}

// ValidateCompatibility checks the server meets the minimum requirements for
// logical replication: PostgreSQL 15+, wal_level=logical, and at least one
// replication slot and WAL sender available. All failures wrap errs.ErrDbSetup.
func ValidateCompatibility(ctx context.Context, conn *connection.Connection) error {
	cfg, err := GetConfig(ctx, conn)
	if err != nil {
		return err
	}

	if cfg.Version < 15 {
		return fmt.Errorf("unsupported version: PostgreSQL version should be 15 or higher, got %d: %w", cfg.Version, errs.ErrDbSetup)
	}

	if cfg.WalLevel != WalLevelLogical {
		return fmt.Errorf("unsupported wal_level: should be logical, got %s: %w", cfg.WalLevel, errs.ErrDbSetup)
	}

	if cfg.MaxReplicationSlots < 1 {
		return fmt.Errorf("max_replication_slots too low: max_replication_slots should be at least 1, got %d: %w", cfg.MaxReplicationSlots, errs.ErrDbSetup)
	}

	if cfg.MaxWalSenders < 1 {
		return fmt.Errorf("max_wal_senders too low: max_wal_senders should be at least 1, got %d: %w", cfg.MaxWalSenders, errs.ErrDbSetup)
	}

	return nil
}

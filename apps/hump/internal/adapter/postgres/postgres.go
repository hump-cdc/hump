// Package postgres is the CDC source adapter: it bootstraps the publication and
// replication slot, then decodes the logical replication stream into row changes.
package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/config.sql
var configSql string

// closeTimeout caps how long shutdown waits for one session to close.
const closeTimeout = 5 * time.Second

// Health reports the liveness of one part of the adapter to the outside world.
type Health interface {
	MarkHealthy()
	MarkUnhealthy(err error)
}

// Postgres is a database.Database backed by a single PostgreSQL source.
type Postgres struct {
	databaseUrl     string
	dbHost          string
	dbPort          uint16
	dbName          string
	cfg             config.Configuration
	publicationName string
	slotName        string
	gate            *walGate
	snapshot        *snapshot
	healthPostgres  Health
	healthWal       Health
	metrics         *metrics
}

// New returns a Postgres adapter replicating cfg's tables through the given
// publication and slot.
func New(databaseUrl string, cfg config.Configuration, publicationName string, slotName string, healthPostgres Health, healthWal Health, metricsFactory promauto.Factory) *Postgres {
	p := &Postgres{
		databaseUrl:     databaseUrl,
		cfg:             cfg,
		publicationName: publicationName,
		slotName:        slotName,
		healthPostgres:  healthPostgres,
		healthWal:       healthWal,
		gate:            newWalGate(),
	}

	// Parsed once here rather than per getter: the getters serve a web UI
	// handler, and ParseConfig reads PG* environment variables and may read .pgpass.
	if _, host, port, name, err := newConnectionOptions(databaseUrl).GetConnectionConfig(); err == nil {
		p.dbHost = host
		p.dbPort = port
		p.dbName = string(name)
	}

	// The collectors register on the factory, which panics on a duplicate, so
	// they are built once here rather than per Listen.
	p.metrics = newMetrics(p.GetDbName(), metricsFactory)
	p.snapshot = newSnapshot(databaseUrl, p.gate, p.metrics)

	return p
}

// configure validates the database, introspects the schema, and ensures the
// publication and replication slot exist. conn is borrowed for the lifetime of
// the returned introspector, so the caller must keep it open until then.
func (p *Postgres) configure(ctx context.Context, conn *connection) (*introspector, error) {
	checkStandardConformingStringsDefault(ctx, p.databaseUrl)

	if err := p.validateCompatibility(ctx, conn); err != nil {
		return nil, fmt.Errorf("validate database compatibility: %w", err)
	}

	registry := newIntrospectorRegistry(p.cfg)
	bs := newBootstrapper(conn)

	// tables arrives already expanded to the configured tables and their
	// partitions, so the bootstrapper needs no catalog lookup of its own.
	introspector := newIntrospector(conn, registry, func(ctx context.Context, _ *connection, tables []domain.TableIdentifier) error {
		return bs.EnsureReplicaIdentityFull(ctx, tables)
	})

	if err := introspector.Invalidate(ctx, nil); err != nil {
		return nil, fmt.Errorf("invalidate introspection: %w", err)
	}

	if err := bs.EnsurePublicationExists(ctx, p.publicationName, registry.List()); err != nil {
		return nil, fmt.Errorf("ensure publication: %w", err)
	}

	bs.WarnMissingReplicaIdentity(ctx, p.publicationName)

	if err := bs.EnsureReplicationSlotExists(ctx, p.slotName); err != nil {
		return nil, fmt.Errorf("ensure replication slot: %w", err)
	}

	return introspector, nil
}

// validateCompatibility checks the server meets the minimum requirements for
// logical replication: PostgreSQL 15+, wal_level=logical, and at least one
// replication slot and WAL sender available. All failures wrap database.ErrDbSetup.
func (p *Postgres) validateCompatibility(ctx context.Context, conn *connection) error {
	results, err := conn.Exec(ctx, configSql)
	if err != nil {
		return err
	}
	if len(results) == 0 || len(results[0].Rows) == 0 {
		return fmt.Errorf("empty database config result: %w", database.ErrDbSetup)
	}

	row := results[0].Rows[0]

	versionNum, err := strconv.Atoi(string(row[0]))
	if err != nil {
		return fmt.Errorf("parse server_version_num: %w", errors.Join(err, database.ErrDbSetup))
	}
	if version := versionNum / 10000; version < 15 {
		return fmt.Errorf("unsupported version: PostgreSQL version should be 15 or higher, got %d: %w", version, database.ErrDbSetup)
	}

	if walLevel := string(row[1]); walLevel != "logical" {
		return fmt.Errorf("unsupported wal_level: should be logical, got %s: %w", walLevel, database.ErrDbSetup)
	}

	maxReplicationSlots, err := strconv.Atoi(string(row[2]))
	if err != nil {
		return fmt.Errorf("parse max_replication_slots: %w", errors.Join(err, database.ErrDbSetup))
	}
	if maxReplicationSlots < 1 {
		return fmt.Errorf("max_replication_slots too low: max_replication_slots should be at least 1, got %d: %w", maxReplicationSlots, database.ErrDbSetup)
	}

	maxWalSenders, err := strconv.Atoi(string(row[3]))
	if err != nil {
		return fmt.Errorf("parse max_wal_senders: %w", errors.Join(err, database.ErrDbSetup))
	}
	if maxWalSenders < 1 {
		return fmt.Errorf("max_wal_senders too low: max_wal_senders should be at least 1, got %d: %w", maxWalSenders, database.ErrDbSetup)
	}

	return nil
}

// closeDetached runs closeFn on a context detached from the caller's, which is
// usually already cancelled by the time shutdown reaches it. On a cancelled one
// pgconn races its deadline handler against the Terminate write instead of
// sending it cleanly.
//
//nolint:contextcheck // detached close: the caller ctx is usually cancelled at shutdown
func closeDetached(closeFn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()

	_ = closeFn(ctx)
}

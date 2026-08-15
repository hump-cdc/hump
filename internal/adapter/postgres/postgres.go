package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/config.sql
var configSql string

// Health reports the liveness of one part of the adapter to the outside world.
type Health interface {
	MarkHealthy()
	MarkUnhealthy(err error)
}

// Postgres is a database.Database backed by a single PostgreSQL source.
type Postgres struct {
	databaseUrl     string
	cfg             config.Configuration
	publicationName string
	slotName        string
	healthPostgres  Health
	healthWal       Health
	metricsFactory  promauto.Factory
}

// New returns a Postgres adapter replicating cfg's tables through the given
// publication and slot.
func New(databaseUrl string, cfg config.Configuration, publicationName string, slotName string, healthPostgres Health, healthWal Health, metricsFactory promauto.Factory) *Postgres {
	return &Postgres{
		databaseUrl:     databaseUrl,
		cfg:             cfg,
		publicationName: publicationName,
		slotName:        slotName,
		healthPostgres:  healthPostgres,
		healthWal:       healthWal,
		metricsFactory:  metricsFactory,
	}
}

// GetDbName returns the database name parsed from the connection string, or an
// empty string when the string is unusable.
func (p *Postgres) GetDbName() string {
	cfg, err := pgconn.ParseConfig(p.databaseUrl)
	if err != nil {
		return ""
	}

	return cfg.Database
}

// GetDbHost returns the host parsed from the connection string, or an empty
// string when the string is unusable.
func (p *Postgres) GetDbHost() string {
	cfg, err := pgconn.ParseConfig(p.databaseUrl)
	if err != nil {
		return ""
	}

	return cfg.Host
}

// GetDbPort returns the port parsed from the connection string, or zero when
// the string is unusable.
func (p *Postgres) GetDbPort() uint16 {
	cfg, err := pgconn.ParseConfig(p.databaseUrl)
	if err != nil {
		return 0
	}

	return cfg.Port
}

// GetPublicationName returns the publication this adapter replicates through.
func (p *Postgres) GetPublicationName() string {
	return p.publicationName
}

// GetSlotName returns the replication slot this adapter streams from.
func (p *Postgres) GetSlotName() string {
	return p.slotName
}

// Listen bootstraps the database over a control connection, then streams WAL
// over a second, replication-enabled one, dispatching decoded rows to handler.
// It blocks until ctx is cancelled or the stream fails.
func (p *Postgres) Listen(ctx context.Context, handler database.ProcessMessageHandler) error {
	conn, err := newConnection(p.databaseUrl)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	introspector, err := p.configure(ctx, conn)
	if err != nil {
		return fmt.Errorf("unable to configure to database: %w", err)
	}

	p.healthPostgres.MarkHealthy()

	replicationConnection, err := newConnection(p.databaseUrl, withReplication(true))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	w, err := newWal(ctx, replicationConnection, p.publicationName, p.slotName, introspector, p.healthWal, p.metricsFactory)
	if err != nil {
		return fmt.Errorf("failed to initialize WAL processing: %w", err)
	}

	err = w.Start(ctx, handler)
	// Start usually returns because ctx was cancelled, so shutdown gets its own
	// context to close the replication connection on.
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	defer w.Stop(closeCtx)

	return err
}

// Cleanup removes the publication and replication slot, leaving replica identity
// untouched. The slot is only dropped when inactive; an active one errors so a
// live consumer keeps its retained WAL.
func (p *Postgres) Cleanup(ctx context.Context) error {
	conn, err := newConnection(p.databaseUrl)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	bs := newBootstrapper(conn)

	if err := bs.EnsurePublicationRemoved(ctx, p.publicationName); err != nil {
		return fmt.Errorf("remove publication: %w", err)
	}

	if err := bs.EnsureReplicationSlotRemoved(ctx, p.slotName); err != nil {
		return fmt.Errorf("remove replication slot: %w", err)
	}

	return nil
}

// configure validates the database, introspects the schema, and ensures the
// publication and replication slot exist. conn is borrowed for the lifetime of
// the returned introspector, so the caller must keep it open until then.
func (p *Postgres) configure(ctx context.Context, conn *connection) (*introspector, error) {
	if err := p.validateCompatibility(ctx, conn); err != nil {
		return nil, fmt.Errorf("validate database compatibility: %w", err)
	}

	registry := newIntrospectorRegistry(p.cfg)
	bs := newBootstrapper(conn)

	introspector := newIntrospector(conn, registry, func(ctx context.Context, _ *connection, tables []domain.TableIdentifier) error {
		return bs.EnsureReplicaIdentityFull(ctx, tables)
	})

	if err := introspector.Invalidate(ctx, nil); err != nil {
		return nil, fmt.Errorf("invalidate introspection: %w", err)
	}

	if err := bs.EnsurePublicationExists(ctx, p.publicationName, registry.List()); err != nil {
		return nil, fmt.Errorf("ensure publication: %w", err)
	}

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

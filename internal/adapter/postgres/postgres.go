package postgres

import (
	"context"
	"fmt"

	"hump/config"
	"hump/internal/adapter/postgres/catalog"
	"hump/internal/adapter/postgres/connection"
	"hump/internal/adapter/postgres/introspection"
	"hump/internal/adapter/postgres/replication"
	"hump/internal/adapter/postgres/schema"
	"hump/internal/types/introspector"
)

// Connection primitives
type (
	Connector  = connection.Connector
	Connection = connection.Connection
)

// Domain types
type (
	TableRef             = introspector.TableRef
	Partition            = schema.Partition
	TableInfo            = schema.TableInfo
	TableType            = schema.TableType
	TablesWithPartitions = schema.TablesWithPartitions
)

// Catalog types
type (
	Config              = catalog.Config
	Summary             = catalog.Summary
	ReplicationSlotInfo = catalog.ReplicationSlotInfo
	PublicationInfo     = catalog.PublicationInfo
)

// Introspector types
type (
	TableRegistry = introspection.TableRegistry
	Introspector  = introspection.Introspector
)

// Constructors and functions
var (
	NewConnector = connection.New

	GetConfig               = catalog.GetConfig
	GetTablesWithPartitions = catalog.GetTablesWithPartitions
	GetTableByPartition     = catalog.GetTableByPartition
	IntrospectTables        = catalog.IntrospectTables
	GetSummary              = catalog.GetSummary
	PrintSummary            = catalog.PrintSummary

	NewTableRegistry = introspection.NewTableRegistry
)

// Configure validates the database, builds the table registry from cfg,
// runs the introspection, and ensures the publication and replication slot
// exist. It opens a single connection from connector and closes it before
// returning.
func Configure(
	ctx context.Context,
	connector *Connector,
	cfg config.Configuration,
	publicationName string,
	slotName string,
) (*TableRegistry, *Introspector, error) {
	conn, err := connector.CreateConnection(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("open connection: %w", err)
	}
	defer conn.Close(ctx)

	if err := catalog.ValidateCompatibility(ctx, conn); err != nil {
		return nil, nil, fmt.Errorf("validate database compatibility: %w", err)
	}

	registry := introspection.RegistryFromConfig(cfg)

	introspector := introspection.NewIntrospector(connector, registry, func(ctx context.Context, conn *Connection, tables []TableRef) error {
		return replication.EnsureReplicaIdentityFull(ctx, conn, tables)
	})

	if err := introspector.Invalidate(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("invalidate introspection: %w", err)
	}

	if err := replication.EnsurePublicationExists(ctx, conn, publicationName, registry.List()); err != nil {
		return nil, nil, fmt.Errorf("ensure publication: %w", err)
	}

	if err := replication.EnsureReplicationSlotExists(ctx, conn, slotName); err != nil {
		return nil, nil, fmt.Errorf("ensure replication slot: %w", err)
	}

	return registry, introspector, nil
}

// Cleanup removes the publication and replication slot created for the service,
// then returns. It does NOT alter replica identity on any table. It opens a
// single connection from connector and closes it before returning.
//
// The replication slot is only dropped when inactive; an active slot (still held
// by a running instance) yields an error so its retained WAL is not pulled out
// from under a live consumer.
func Cleanup(
	ctx context.Context,
	connector *Connector,
	publicationName string,
	slotName string,
) error {
	conn, err := connector.CreateConnection(ctx)
	if err != nil {
		return fmt.Errorf("open connection: %w", err)
	}
	defer conn.Close(ctx)

	if err := replication.EnsurePublicationRemoved(ctx, conn, publicationName); err != nil {
		return fmt.Errorf("remove publication: %w", err)
	}

	if err := replication.EnsureReplicationSlotRemoved(ctx, conn, slotName); err != nil {
		return fmt.Errorf("remove replication slot: %w", err)
	}

	return nil
}

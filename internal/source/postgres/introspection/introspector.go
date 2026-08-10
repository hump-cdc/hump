// Package introspection keeps the in-memory view of configured tables in
// sync with the live PostgreSQL schema. It resolves partition tables to
// their configured parent, lazily re-introspects when unknown tables are
// seen at runtime, and ensures replica identity is set on replicated tables.
package introspection

import (
	"context"
	"sync"

	"hump/internal/domain"
	"hump/internal/source/postgres/catalog"
	"hump/internal/source/postgres/connection"
	"hump/internal/types/config"
	"hump/internal/types/introspector"
)

// Introspector manages in-memory table configuration in response to schema
// changes (e.g., new partitions) and ensures replica identity is set for
// replicated tables.
type Introspector struct {
	connector             *connection.Connector
	registry              *TableRegistry
	mu                    sync.Mutex
	initialized           bool
	ignoredTables         map[string]struct{}
	ensureReplicaIdentity func(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) error
}

// NewIntrospector returns an Introspector backed by connector and registry.
// ensureReplicaIdentity is invoked after every successful introspection to
// keep replicated tables at REPLICA IDENTITY FULL.
func NewIntrospector(
	connector *connection.Connector,
	registry *TableRegistry,
	ensureReplicaIdentity func(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) error,
) *Introspector {
	return &Introspector{
		connector:             connector,
		registry:              registry,
		ignoredTables:         make(map[string]struct{}),
		ensureReplicaIdentity: ensureReplicaIdentity,
	}
}

// Invalidate re-introspects all configured tables and refreshes the
// partition→parent mapping in the registry. When table is non-nil, the
// service first checks whether that table is a known partition: if it is
// not, the call is a no-op (skip introspection for unrelated tables).
func (i *Introspector) Invalidate(ctx context.Context, table *introspector.TableRef) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	conn, err := i.connector.CreateConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	return i.invalidateLocked(ctx, conn, table)
}

// invalidateLocked performs the introspection refresh while the caller holds
// i.mu. See Invalidate for the semantics of the table argument.
func (i *Introspector) invalidateLocked(ctx context.Context, conn *connection.Connection, table *introspector.TableRef) error {
	// if table is provided, trying to resolve it by partition
	// this parameter is provided when we cannot find partition
	// in the previous interspection result, so trying to re-invalidate
	if table != nil {
		partition, err := catalog.GetTableByPartition(ctx, conn, table.Schema, table.Name, i.registry.List())
		if err != nil {
			return err
		}

		// table is not partitioned, so skipping introspection
		if partition == nil {
			return nil
		}
	}

	introspection, err := catalog.IntrospectTables(ctx, conn, i.registry.List())
	if err != nil {
		return err
	}

	i.registry.UpdatePartitions(introspection.Partitions)

	if err := i.ensureReplicaIdentity(ctx, conn, i.registry.ReplicaIdentityFullTables()); err != nil {
		return err
	}

	if table == nil {
		i.initialized = true
	}

	return nil
}

// ResolveTableConfiguration returns the resolved table reference and the
// sinks configured for schema/table. When the input is a partition, the
// returned TableRef points to the parent table. On first call it lazily runs
// a full introspection. If no configuration is found it re-introspects once
// (to pick up partitions created after startup), and ignores future lookups
// for the same key if still not found.
func (i *Introspector) ResolveTableConfiguration(ctx context.Context, schemaName domain.TableSchema, name domain.TableName) (introspector.TableRef, []config.SinkConfig, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// initialize on first call
	if !i.initialized {
		// db connection
		conn, err := i.connector.CreateConnection(ctx)
		if err != nil {
			return introspector.TableRef{}, nil, err
		}
		defer conn.Close(ctx)

		// invalidate tables to populate registry
		if err := i.invalidateLocked(ctx, conn, nil); err != nil {
			return introspector.TableRef{}, nil, err
		}
	}

	// resolve table configuration from registry
	if ref, sinks := i.registry.Resolve(schemaName, name); sinks != nil {
		return ref, sinks, nil
	}

	// if not found, check whether table is ignored (e.g., we already tried to find it after previous introspection and it was not found)
	key := string(schemaName) + "." + string(name)
	if _, ignored := i.ignoredTables[key]; ignored {
		return introspector.TableRef{}, nil, nil
	}

	// if not found and not ignored, re-invalidate registry (e.g., to pick up new partitions created after startup)
	conn, err := i.connector.CreateConnection(ctx)
	if err != nil {
		return introspector.TableRef{}, nil, err
	}
	defer conn.Close(ctx)
	if err := i.invalidateLocked(ctx, conn, &introspector.TableRef{Schema: schemaName, Name: name}); err != nil {
		return introspector.TableRef{}, nil, err
	}

	ref, sinks := i.registry.Resolve(schemaName, name)
	if sinks == nil {
		i.ignoredTables[key] = struct{}{}
	}

	return ref, sinks, nil
}

package replication

import (
	"context"
	_ "embed"
	"fmt"

	"hump/internal/source/postgres/catalog"
	"hump/internal/source/postgres/connection"
	"hump/internal/source/postgres/schema"
	"hump/internal/types/introspector"
)

//go:embed sql/replica_identity.sql
var replicaIdentitySql string

// EnsureReplicaIdentityFull sets REPLICA IDENTITY FULL on every table that
// needs it for logical replication. For each configured table it includes:
//
//   - the parent table itself (regular, partitioned, hypertable);
//   - all partition tables, except for hypertables — TimescaleDB chunks inherit
//     replica identity from the hypertable, and altering chunks directly is
//     unsupported.
//
// Tables already at REPLICA IDENTITY FULL are skipped without issuing ALTER.
func EnsureReplicaIdentityFull(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) error {
	info, err := catalog.GetTablesWithPartitions(ctx, conn, tables)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	var targets []introspector.TableRef
	add := func(s, name string) {
		key := s + "." + name
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		targets = append(targets, introspector.TableRef{Schema: s, Name: name})
	}

	for schemaName, byName := range info {
		for name, t := range byName {
			add(schemaName, name)
			if t.Type == schema.TableTypeHypertable {
				continue
			}
			for _, p := range t.Partitions {
				if p.Schema == "" || p.Table == "" {
					continue
				}
				add(p.Schema, p.Table)
			}
		}
	}

	for _, t := range targets {
		if err := setReplicaIdentityFull(ctx, conn, t.Schema, t.Name); err != nil {
			return err
		}
	}

	return nil
}

// setReplicaIdentityFull issues ALTER TABLE ... REPLICA IDENTITY FULL for the
// given table. The embedded SQL is a no-op when the table is already FULL.
func setReplicaIdentityFull(ctx context.Context, conn *connection.Connection, schemaName, name string) error {
	qualifiedLiteral := connection.QuoteLiteral(schemaName + "." + name)
	qualifiedIdentifier := connection.QuoteIdentifier(schemaName) + "." + connection.QuoteIdentifier(name)

	sql := fmt.Sprintf(replicaIdentitySql, qualifiedLiteral, qualifiedIdentifier)

	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("set replica identity full on %s.%s: %w", schemaName, name, err)
	}

	return nil
}

// Package replication manages the PostgreSQL server-side state required for
// logical replication: the publication that lists replicated tables, the
// replication slot consumed by the service, and REPLICA IDENTITY FULL on
// every table whose changes must be captured.
package replication

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"hump/internal/source/postgres/catalog"
	"hump/internal/source/postgres/connection"
	"hump/internal/source/postgres/schema"
	"hump/internal/types/introspector"
)

//go:embed sql/publication.sql
var publicationSql string

//go:embed sql/publication_remove.sql
var publicationRemoveSql string

// EnsurePublicationExists ensures the logical publication required for the
// service exists in PostgreSQL and includes all configured tables.
//
//   - If the publication does not exist, it is created with the configured tables.
//   - If it exists, its set of published tables is updated to match the
//     desired configuration.
//   - The publish_via_partition_root option is always set to true.
//
// Hypertable partitions are published via "TABLES IN SCHEMA <schema>"
// because chunk tables live in TimescaleDB-managed schemas. Regular and
// declaratively-partitioned tables are published by name.
func EnsurePublicationExists(ctx context.Context, conn *connection.Connection, publicationName string, tables []introspector.TableRef) error {
	info, err := catalog.GetTablesWithPartitions(ctx, conn, tables)
	if err != nil {
		return err
	}

	publication := buildPublicationClause(info)

	pubIdent := connection.QuoteIdentifier(publicationName)
	pubLit := connection.QuoteLiteral(publicationName)

	sql := fmt.Sprintf(publicationSql, pubLit, pubIdent, publication)

	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("ensure publication %q: %w", publicationName, err)
	}

	return nil
}

// EnsurePublicationRemoved drops the publication if it exists. It is a no-op
// when the publication is absent.
func EnsurePublicationRemoved(ctx context.Context, conn *connection.Connection, publicationName string) error {
	sql := fmt.Sprintf(publicationRemoveSql, connection.QuoteIdentifier(publicationName))

	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("remove publication %q: %w", publicationName, err)
	}

	return nil
}

// buildPublicationClause renders the FOR ... portion of CREATE/ALTER
// PUBLICATION. Regular and declaratively-partitioned tables are listed by
// qualified name; hypertable chunks are covered by TABLES IN SCHEMA clauses
// because their schemas are TimescaleDB-managed.
func buildPublicationClause(info schema.TablesWithPartitions) string {
	tableSet := make(map[string]struct{})
	schemaSet := make(map[string]struct{})

	for schemaName, byName := range info {
		for name, t := range byName {
			qualified := connection.QuoteIdentifier(string(schemaName)) + "." + connection.QuoteIdentifier(string(name))
			switch t.Type {
			case schema.TableTypeRegular, schema.TableTypePartitioned:
				tableSet[qualified] = struct{}{}
			case schema.TableTypeHypertable:
				tableSet[qualified] = struct{}{}
				for _, p := range t.Partitions {
					if p.Schema == "" {
						continue
					}
					schemaSet[string(p.Schema)] = struct{}{}
				}
			}
		}
	}

	tableList := sortedKeys(tableSet)
	schemaList := sortedKeys(schemaSet)

	var parts []string
	if len(tableList) > 0 {
		parts = append(parts, "TABLE "+strings.Join(tableList, ", "))
	}
	for _, s := range schemaList {
		parts = append(parts, "TABLES IN SCHEMA "+connection.QuoteIdentifier(s))
	}

	return strings.Join(parts, ", ")
}

// sortedKeys returns the keys of m in ascending order, used to produce
// deterministic SQL across invocations.
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

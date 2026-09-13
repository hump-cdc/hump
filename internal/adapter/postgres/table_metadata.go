package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/table_metadata_tables_timescale.sql
var tableMetadataTablesTimescaleSql string

//go:embed sql/table_metadata_tables_no_timescale.sql
var tableMetadataTablesNoTimescaleSql string

//go:embed sql/table_metadata_has_timescale.sql
var tableMetadataHasTimescaleSql string

// tableMetadataLoader reads the catalog layout of configured tables over conn.
// It is not safe for concurrent use: the timescaledb probe is cached without a
// lock, so an owner shared across goroutines must serialize its calls.
type tableMetadataLoader struct {
	connection *connection
	// newTypeMap builds the pgtype map that encodes array bind parameters. A
	// fresh map per call keeps its encode-plan cache free of concurrent writes;
	// tests replace it with one that refuses to encode.
	newTypeMap func() *pgtype.Map
	// timescale caches the timescaledb extension probe, nil until it has run:
	// the extension cannot be installed or dropped while a session is live.
	timescale *bool
}

// newTableMetadataLoader returns a loader backed by conn. conn is borrowed, not
// owned: the caller must keep it open for the lifetime of the loader.
func newTableMetadataLoader(conn *connection) *tableMetadataLoader {
	return &tableMetadataLoader{connection: conn, newTypeMap: pgtype.NewMap}
}

// getTablesMetadata returns the given tables with their partition layout.
// Returns database.ErrNoTablesConfigured when tables is empty and
// database.ErrTablesNotFound, listing every one, when any table is missing.
func (l *tableMetadataLoader) getTablesMetadata(ctx context.Context, tables []domain.TableIdentifier) (tablesMetadata, error) {
	if len(tables) == 0 {
		return nil, database.ErrNoTablesConfigured
	}

	timescale, err := l.hasTimescaleDB(ctx)
	if err != nil {
		return nil, err
	}

	sql := tableMetadataTablesNoTimescaleSql
	if timescale {
		sql = tableMetadataTablesTimescaleSql
	}

	params, err := tableArrayParams(l.newTypeMap(), tables)
	if err != nil {
		return nil, err
	}

	queryResult, err := l.connection.ExecParams(ctx, sql, params, nil)
	if err != nil {
		return nil, fmt.Errorf("get tables with partitions: %w", err)
	}

	result := make(tablesMetadata)
	for _, values := range queryResult.Rows {
		if len(values) < 5 {
			return nil, fmt.Errorf("get tables with partitions: expected 5 columns, got %d", len(values))
		}

		table := domain.TableIdentifier{Schema: domain.TableSchema(values[0]), Name: domain.TableName(values[1])}
		kind := parseTableType(string(values[2]))
		partitionSchema := values[3]
		partitionTable := values[4]

		info, ok := result[table]
		if !ok {
			info = &tableMetadata{Type: kind}
			result[table] = info
		}

		if partitionSchema != nil {
			p := domain.TableIdentifier{Schema: domain.TableSchema(partitionSchema)}
			if partitionTable != nil {
				p.Name = domain.TableName(partitionTable)
			}
			info.Partitions = append(info.Partitions, p)
		}
	}

	var missing []string
	for _, t := range tables {
		if _, ok := result[t]; !ok {
			missing = append(missing, string(t.Schema)+"."+string(t.Name))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", database.ErrTablesNotFound, strings.Join(missing, ", "))
	}

	return result, nil
}

// hasTimescaleDB reports whether the timescaledb extension is installed, probing
// once per loader. Failures are not cached and an empty result wraps
// database.ErrDbSetup.
func (l *tableMetadataLoader) hasTimescaleDB(ctx context.Context) (bool, error) {
	if l.timescale != nil {
		return *l.timescale, nil
	}

	results, err := l.connection.Exec(ctx, tableMetadataHasTimescaleSql)
	if err != nil {
		return false, err
	}
	if len(results) == 0 || len(results[0].Rows) == 0 {
		return false, fmt.Errorf("empty timescaledb extension result: %w", database.ErrDbSetup)
	}

	timescale := string(results[0].Rows[0][0]) == "t"
	l.timescale = &timescale

	return timescale, nil
}

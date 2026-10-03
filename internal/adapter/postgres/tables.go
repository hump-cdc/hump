package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/tables_columns_no_timescale.sql
var tablesColumnsNoTimescaleSql string

//go:embed sql/tables_columns_timescale.sql
var tablesColumnsTimescaleSql string

// tablesColumnsWidth is how many values each row of the columns query carries.
const tablesColumnsWidth = 10

// tablesColumn is one column of a described table, with the traits suggestWhere
// ranks it by.
type tablesColumn struct {
	database.TableColumn
	TimeKey    bool // TimeKey marks the time dimension of a hypertable or the range partitioning key.
	NowDefault bool // NowDefault marks a column defaulting to the current time.
	Indexed    bool // Indexed marks a column that leads some index.
}

// GetTables describes the given tables, in the order given, each with its
// columns and a suggested snapshot filter. Tables missing from the database are
// skipped. Each call runs on its own session.
func (p *Postgres) GetTables(ctx context.Context, tables []domain.TableIdentifier) ([]database.TableDescription, error) {
	if len(tables) == 0 {
		return []database.TableDescription{}, nil
	}

	conn, err := newConnection(p.databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer closeDetached(conn.Close)

	loader := newTableMetadataLoader(conn)

	timescale, err := loader.hasTimescaleDB(ctx)
	if err != nil {
		return nil, err
	}

	sql := tablesColumnsNoTimescaleSql
	if timescale {
		sql = tablesColumnsTimescaleSql
	}

	params, err := tableArrayParams(loader.newTypeMap(), tables)
	if err != nil {
		return nil, err
	}

	result, err := conn.ExecParams(ctx, sql, params, nil)
	if err != nil {
		return nil, fmt.Errorf("get table columns: %w", err)
	}

	return describeTables(tables, result.Rows)
}

// describeTables groups the rows of the columns query by table and returns the
// tables found, in the order of tables.
func describeTables(tables []domain.TableIdentifier, rows [][][]byte) ([]database.TableDescription, error) {
	type described struct {
		kind    database.TableType
		columns []tablesColumn
	}

	found := make(map[domain.TableIdentifier]*described)
	for _, values := range rows {
		if len(values) < tablesColumnsWidth {
			return nil, fmt.Errorf("get table columns: expected %d columns, got %d", tablesColumnsWidth, len(values))
		}

		table := domain.TableIdentifier{Schema: domain.TableSchema(values[0]), Name: domain.TableName(values[1])}

		d, ok := found[table]
		if !ok {
			d = &described{kind: database.TableType(values[2])}
			found[table] = d
		}

		d.columns = append(d.columns, tablesColumn{
			TableColumn: database.TableColumn{
				Name:       string(values[3]),
				Type:       string(values[4]),
				Nullable:   isTrue(values[5]),
				PrimaryKey: isTrue(values[6]),
			},
			TimeKey:    isTrue(values[7]),
			NowDefault: isTrue(values[8]),
			Indexed:    isTrue(values[9]),
		})
	}

	result := make([]database.TableDescription, 0, len(tables))
	for _, table := range tables {
		d, ok := found[table]
		if !ok {
			continue
		}

		columns := make([]database.TableColumn, 0, len(d.columns))
		for _, c := range d.columns {
			columns = append(columns, c.TableColumn)
		}

		result = append(result, database.TableDescription{
			Table:   table,
			Type:    d.kind,
			Columns: columns,
			Where:   suggestWhere(d.columns),
		})
	}

	return result, nil
}

// isTrue reports whether value is a text-format boolean true.
func isTrue(value []byte) bool {
	return string(value) == "t"
}

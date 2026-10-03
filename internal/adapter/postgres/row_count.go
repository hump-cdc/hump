package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/row_count_table_exists.sql
var rowCountTableExistsSql string

//go:embed sql/row_count_estimate.sql
var rowCountEstimateSql string

//go:embed sql/row_count_exact.sql
var rowCountExactSql string

// rowCountTimeout caps each row count statement, the planner estimate included:
// the planner folds immutable calls on constants, so even EXPLAIN can run long.
const rowCountTimeout = 2 * time.Second

// errEmptyRowCount indicates a row count statement returned no value.
var errEmptyRowCount = errors.New("row count statement returned no value")

// GetRowCount validates where and checks that table exists, then streams the
// planner's estimate of the matching rows followed by the exact count. A failed
// estimate is sent as an error and ends the stream; a failed or timed out exact
// count is dropped. Each call runs on its own session, so calls may overlap.
func (p *Postgres) GetRowCount(ctx context.Context, table domain.TableIdentifier, where string) (<-chan database.RowCount, error) {
	condition, err := parseWhere(where)
	if err != nil {
		return nil, fmt.Errorf("parse where: %w", errors.Join(err, database.ErrInvalidWhere))
	}

	if condition != "" {
		condition = " WHERE " + condition
	}

	conn, err := newConnection(p.databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	qualified := qualifiedTableName(table)

	exists, err := rowCountTableExists(ctx, conn, qualified)
	if err != nil {
		closeDetached(conn.Close)
		return nil, fmt.Errorf("check table %s: %w", qualified, err)
	}

	if !exists {
		closeDetached(conn.Close)
		return nil, fmt.Errorf("table %s: %w", qualified, database.ErrTableNotFound)
	}

	// Buffered for both results, so the sender never blocks on a reader that left.
	counts := make(chan database.RowCount, 2)

	go func() {
		defer close(counts)
		defer closeDetached(conn.Close)

		estimate, err := rowCountEstimate(ctx, conn, qualified, condition)
		if err != nil {
			counts <- database.RowCount{Error: err}
			return
		}
		counts <- database.RowCount{Num: estimate, Rough: true}

		if exact, err := rowCountExact(ctx, conn, qualified, condition); err == nil {
			counts <- database.RowCount{Num: exact}
		}
	}()

	return counts, nil
}

// rowCountTableExists reports whether qualified resolves to a relation.
func rowCountTableExists(ctx context.Context, conn *connection, qualified string) (bool, error) {
	result, err := conn.ExecParams(ctx, rowCountTableExistsSql, []param{textParam(qualified)}, nil)
	if err != nil {
		return false, err
	}

	value, err := rowCountValue([]*pgconn.Result{result})
	if err != nil {
		return false, err
	}

	return string(value) == "t", nil
}

// rowCountEstimate returns the planner's row estimate for the filtered table,
// read from the top plan node, which already sums partitions and chunks.
func rowCountEstimate(ctx context.Context, conn *connection, qualified, condition string) (uint64, error) {
	value, err := rowCountQuery(ctx, conn, rowCountEstimateSql, qualified, condition)
	if err != nil {
		return 0, err
	}

	var plans []struct {
		Plan struct {
			Rows float64 `json:"Plan Rows"`
		} `json:"Plan"`
	}
	if err := json.Unmarshal(value, &plans); err != nil {
		return 0, fmt.Errorf("decode plan: %w", err)
	}

	if len(plans) == 0 {
		return 0, errEmptyRowCount
	}

	return uint64(plans[0].Plan.Rows), nil
}

// rowCountExact returns the exact number of rows matching the filter.
func rowCountExact(ctx context.Context, conn *connection, qualified, condition string) (uint64, error) {
	value, err := rowCountQuery(ctx, conn, rowCountExactSql, qualified, condition)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(string(value), 10, 64)
}

// rowCountQuery runs a row count template as one simple-protocol message, so a
// reconnect cannot replay the statement outside its read-only transaction and
// timeout, and returns the value it selected.
func rowCountQuery(ctx context.Context, conn *connection, template, qualified, condition string) ([]byte, error) {
	results, err := conn.Exec(ctx, fmt.Sprintf(template, rowCountTimeout.Milliseconds(), qualified, condition))
	if err != nil {
		return nil, err
	}

	return rowCountValue(results)
}

// rowCountValue returns the first column of the first row any result carries,
// skipping the empty results of the transaction control statements.
func rowCountValue(results []*pgconn.Result) ([]byte, error) {
	for _, result := range results {
		if len(result.Rows) > 0 && len(result.Rows[0]) > 0 {
			return result.Rows[0][0], nil
		}
	}

	return nil, errEmptyRowCount
}

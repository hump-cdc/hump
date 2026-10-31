package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"hump/internal/domain"
	"hump/internal/port/database"
)

//go:embed sql/snapshot_export.sql
var snapshotExportSql string

//go:embed sql/snapshot_layout_no_timescale.sql
var snapshotLayoutNoTimescaleSql string

//go:embed sql/snapshot_layout_timescale.sql
var snapshotLayoutTimescaleSql string

//go:embed sql/snapshot_read.sql
var snapshotReadSql string

const (
	// snapshotLockTimeout caps the wait for the snapshot's table locks, which queue
	// behind any DDL holding the tables.
	snapshotLockTimeout = 5 * time.Second
	// snapshotChunkPages is how many heap pages one COPY of a snapshot reads.
	snapshotChunkPages = 1024
)

// errEmptySnapshotId indicates the snapshot export returned no id.
var errEmptySnapshotId = errors.New("snapshot export returned no id")

// snapshotHolder is the session whose open transaction keeps an exported
// snapshot importable and its tables locked against rewrites.
type snapshotHolder struct {
	conn *connection
	id   string
}

// snapshotLayout is what a snapshot read needs to know about a table.
type snapshotLayout struct {
	leaves  []snapshotLeaf
	columns []snapshotColumn
}

// snapshotLeaf is one relation that stores rows of a snapshotted table.
type snapshotLeaf struct {
	table domain.TableIdentifier
	pages uint64
}

// snapshotColumn is one column a snapshot read selects.
type snapshotColumn struct {
	name string
	oid  uint32
}

// snapshot exports snapshots of one database, one at a time, and reads tables
// from them. Unlike wal it lives as long as the adapter: a snapshot may be held
// across Listen calls.
type snapshot struct {
	databaseUrl string
	gate        *walGate // gate pauses WAL delivery while a snapshot is active.
	metrics     *metrics
	mu          sync.Mutex
	holder      *snapshotHolder // holder is the active snapshot; nil when none is.
}

// newSnapshot returns a snapshot with none active, pausing WAL through gate.
// m is owned by the caller.
func newSnapshot(databaseUrl string, gate *walGate, m *metrics) *snapshot {
	return &snapshot{
		databaseUrl: databaseUrl,
		gate:        gate,
		metrics:     m,
	}
}

// ValidateWhere reports whether where is a row filter snapshotWhereParser accepts.
func (s *snapshot) ValidateWhere(where string) error {
	if _, err := newSnapshotWhereParser().Parse(where); err != nil {
		return fmt.Errorf("parse where: %w", errors.Join(err, database.ErrInvalidWhere))
	}

	return nil
}

// Start pauses WAL delivery, then opens a read-only transaction on a
// session of its own that takes ACCESS SHARE locks on tables, their partitions
// and chunks included, and exports its snapshot. While a snapshot is active it
// returns the same id.
func (s *snapshot) Start(ctx context.Context, tables []domain.TableIdentifier) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.holder != nil {
		return s.holder.id, nil
	}

	if len(tables) == 0 {
		return "", fmt.Errorf("no tables to snapshot: %w", database.ErrSnapshot)
	}

	if err := s.gate.pause(ctx); err != nil {
		s.gate.resume()
		return "", fmt.Errorf("pause WAL: %w", errors.Join(err, database.ErrSnapshot))
	}

	holder, err := s.establishSnapshot(ctx, tables)
	if err != nil {
		s.gate.resume()
		return "", err
	}

	s.holder = holder

	return holder.id, nil
}

// Finish commits the snapshot's transaction, closes its session and
// resumes WAL delivery. It is a no-op without an active snapshot.
func (s *snapshot) Finish(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	holder := s.holder
	if holder == nil {
		return nil
	}

	s.holder = nil
	defer s.gate.resume()
	defer closeDetached(holder.conn.Close)

	if _, err := holder.conn.Exec(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit snapshot: %w", errors.Join(err, database.ErrSnapshot))
	}

	return nil
}

// Read streams matching user records and their physical leaf identity separately.
// Each leaf is read in ctid page chunks, with processed rows observed as inserts of table.
func (s *snapshot) Read(ctx context.Context, snapshotID string, table domain.TableIdentifier, where string, onRow func(domain.DbRecord, database.SnapshotRowMetadata) error) error {
	whereParser := newSnapshotWhereParser()
	condition, err := whereParser.Parse(where)
	if err != nil {
		return fmt.Errorf("parse where: %w", errors.Join(err, database.ErrInvalidWhere))
	}

	if condition == "" {
		condition = "true"
	}

	conn, err := newConnection(s.databaseUrl)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer closeDetached(conn.Close)

	layout, err := s.readSnapshotLayout(ctx, conn, snapshotID, table)
	if err != nil {
		return err
	}

	if len(layout.columns) == 0 {
		return fmt.Errorf("table %s has no columns to read: %w", qualifiedTableName(table), database.ErrSnapshot)
	}

	whereParser.columnQualifier = "snapshot_row"
	whereParser.tableQualifiers = map[string]bool{string(table.Name): true}
	for _, leaf := range layout.leaves {
		whereParser.tableQualifiers[string(leaf.table.Name)] = true
	}
	if condition != "true" {
		condition, err = whereParser.Parse(where)
		if err != nil {
			return fmt.Errorf("parse where: %w", errors.Join(err, database.ErrInvalidWhere))
		}
	}

	names := make([]string, len(layout.columns))
	for i, column := range layout.columns {
		names[i] = "snapshot_row." + quoteIdentifier(column.name)
	}
	selectList := strings.Join(names, ", ")

	typeMap := newValueTypeMap()
	observeRow := func(record domain.DbRecord, metadata database.SnapshotRowMetadata) error {
		start := time.Now()
		if err := onRow(record, metadata); err != nil {
			return err
		}

		s.metrics.observeRow(domain.OperationTypeInsert, table, time.Since(start))

		return nil
	}

	for _, leaf := range layout.leaves {
		parser := newSnapshotExportParser(leaf.table, layout.columns, typeMap, observeRow)
		for _, rng := range s.snapshotRanges(leaf.pages) {
			// A raw session: a transparent reconnect would replay a chunk whose rows
			// were already delivered, so a dead session ends the read instead.
			session, err := conn.Raw(ctx)
			if err != nil {
				return fmt.Errorf("unable to connect to database: %w", err)
			}

			sql := fmt.Sprintf(snapshotReadSql, quoteLiteral(snapshotID), selectList, qualifiedTableName(leaf.table), rng+"("+condition+")")
			logger.Info("Reading snapshot chunk", "schema", leaf.table.Schema, "table", leaf.table.Name)

			if _, err := session.CopyTo(ctx, parser, sql); err != nil {
				return fmt.Errorf("read %s: %w", qualifiedTableName(leaf.table), errors.Join(err, database.ErrSnapshot))
			}

			if err := parser.finish(); err != nil {
				return fmt.Errorf("read %s: %w", qualifiedTableName(leaf.table), err)
			}
		}
	}

	return nil
}

// establishSnapshot opens the holder session and exports its snapshot. The whole
// transaction goes out as one simple-protocol message, so a reconnect cannot
// split it.
func (s *snapshot) establishSnapshot(ctx context.Context, tables []domain.TableIdentifier) (*snapshotHolder, error) {
	conn, err := newConnection(s.databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	qualified := make([]string, len(tables))
	for i, table := range tables {
		qualified[i] = qualifiedTableName(table)
	}

	results, err := conn.Exec(ctx, fmt.Sprintf(snapshotExportSql, snapshotLockTimeout.Milliseconds(), strings.Join(qualified, ", ")))
	if err != nil {
		closeDetached(conn.Close)
		return nil, fmt.Errorf("export snapshot: %w", errors.Join(err, database.ErrSnapshot))
	}

	id, err := rowCountValue(results)
	if err != nil || len(id) == 0 {
		closeDetached(conn.Close)
		return nil, fmt.Errorf("export snapshot: %w", errors.Join(errEmptySnapshotId, database.ErrSnapshot))
	}

	return &snapshotHolder{conn: conn, id: string(id)}, nil
}

// readSnapshotLayout reads the leaf relations of table and its columns inside
// the snapshot, picking the TimescaleDB flavour of the query when the extension
// is installed.
func (s *snapshot) readSnapshotLayout(ctx context.Context, conn *connection, snapshotID string, table domain.TableIdentifier) (snapshotLayout, error) {
	timescale, err := newTableMetadataLoader(conn).hasTimescaleDB(ctx)
	if err != nil {
		return snapshotLayout{}, err
	}

	queryTemplate := snapshotLayoutNoTimescaleSql
	if timescale {
		queryTemplate = snapshotLayoutTimescaleSql
	}

	results, err := conn.Exec(ctx, fmt.Sprintf(queryTemplate, quoteLiteral(snapshotID), quoteLiteral(qualifiedTableName(table))))
	if err != nil {
		return snapshotLayout{}, fmt.Errorf("read layout of %s: %w", qualifiedTableName(table), errors.Join(err, database.ErrSnapshot))
	}

	// BEGIN, SET TRANSACTION, leaves, columns, COMMIT.
	if len(results) != 5 {
		return snapshotLayout{}, fmt.Errorf("read layout of %s: expected 5 results, got %d: %w", qualifiedTableName(table), len(results), database.ErrSnapshot)
	}

	return s.parseSnapshotLayout(results[2], results[3])
}

// parseSnapshotLayout decodes the leaves and columns results of the layout query.
func (s *snapshot) parseSnapshotLayout(leaves, columns *pgconn.Result) (snapshotLayout, error) {
	var layout snapshotLayout

	for _, row := range leaves.Rows {
		if len(row) < 3 {
			return snapshotLayout{}, fmt.Errorf("read layout: expected 3 leaf columns, got %d: %w", len(row), database.ErrSnapshot)
		}

		pages, err := strconv.ParseUint(string(row[2]), 10, 64)
		if err != nil {
			return snapshotLayout{}, fmt.Errorf("read layout: parse pages: %w", errors.Join(err, database.ErrSnapshot))
		}

		layout.leaves = append(layout.leaves, snapshotLeaf{
			table: domain.TableIdentifier{Schema: domain.TableSchema(row[0]), Name: domain.TableName(row[1])},
			pages: pages,
		})
	}

	for _, row := range columns.Rows {
		if len(row) < 2 {
			return snapshotLayout{}, fmt.Errorf("read layout: expected 2 column fields, got %d: %w", len(row), database.ErrSnapshot)
		}

		oid, err := strconv.ParseUint(string(row[1]), 10, 32)
		if err != nil {
			return snapshotLayout{}, fmt.Errorf("read layout: parse column type: %w", errors.Join(err, database.ErrSnapshot))
		}

		layout.columns = append(layout.columns, snapshotColumn{name: string(row[0]), oid: uint32(oid)})
	}

	return layout, nil
}

// snapshotRanges returns the ctid conditions splitting a relation of pages
// pages into chunks. The last chunk is open-ended, so rows on pages added past
// the size read up front are not missed.
func (s *snapshot) snapshotRanges(pages uint64) []string {
	if pages <= snapshotChunkPages {
		return []string{""}
	}

	ranges := make([]string, 0, pages/snapshotChunkPages+1)
	for from := uint64(0); from < pages; from += snapshotChunkPages {
		to := from + snapshotChunkPages

		var rng strings.Builder
		if from > 0 {
			rng.WriteString("snapshot_row.ctid >= '(" + strconv.FormatUint(from, 10) + ",0)'::tid AND ")
		}
		if to < pages {
			rng.WriteString("snapshot_row.ctid < '(" + strconv.FormatUint(to, 10) + ",0)'::tid AND ")
		}

		ranges = append(ranges, rng.String())
	}

	return ranges
}

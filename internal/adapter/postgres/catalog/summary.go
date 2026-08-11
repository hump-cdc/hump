package catalog

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/adapter/postgres/connection"
	"hump/internal/types/errs"
	"hump/internal/types/introspector"
	"hump/pkg/telemetry/log"
)

//go:embed sql/summary.sql
var summarySql string

//go:embed sql/summary_replica_identities.sql
var replicaIdentitiesSql string

var logger = log.Scope("database")

// ReplicationSlotInfo describes one row of pg_replication_slots: whether the
// slot is currently attached to a walsender, and the textual replication lag.
type ReplicationSlotInfo struct {
	Active bool
	Lag    string
}

// PublicationInfo describes one row of pg_publication: its owner, the
// comma-separated list of tables it publishes, and how many replication slots
// reference it.
type PublicationInfo struct {
	Owner      string
	Tables     string
	TotalSlots int
}

// Summary is a snapshot of replication-related catalog state: server settings,
// replication slots, publications, and per-table replica identity values
// (keyed as "schema.table").
type Summary struct {
	Settings          map[string]string
	ReplicationSlots  map[string]ReplicationSlotInfo
	Publications      map[string]PublicationInfo
	ReplicaIdentities map[string]string
}

// GetSummary runs the summary query plus, when tables is non-empty, a second
// query to fetch replica identities for those tables. All catalog read errors
// wrap errs.ErrDbSetup.
func GetSummary(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) (*Summary, error) {
	results, err := conn.Exec(ctx, summarySql)
	if err != nil {
		return nil, fmt.Errorf("get database summary: %w", errors.Join(err, errs.ErrDbSetup))
	}
	if len(results) < 3 {
		return nil, fmt.Errorf("get database summary: unexpected number of result sets: %w", errs.ErrDbSetup)
	}

	out := &Summary{
		Settings:          make(map[string]string),
		ReplicationSlots:  make(map[string]ReplicationSlotInfo),
		Publications:      make(map[string]PublicationInfo),
		ReplicaIdentities: make(map[string]string),
	}

	for _, row := range results[0].Rows {
		if len(row) < 2 {
			return nil, fmt.Errorf("get database summary: settings row expected 2 columns, got %d: %w", len(row), errs.ErrDbSetup)
		}
		out.Settings[string(row[0])] = string(row[1])
	}

	for _, row := range results[1].Rows {
		if len(row) < 4 {
			return nil, fmt.Errorf("get database summary: replication slots row expected 4 columns, got %d: %w", len(row), errs.ErrDbSetup)
		}
		out.ReplicationSlots[string(row[0])] = ReplicationSlotInfo{
			Active: string(row[1]) == "t",
			Lag:    string(row[3]),
		}
	}

	for _, row := range results[2].Rows {
		if len(row) < 4 {
			return nil, fmt.Errorf("get database summary: publications row expected 4 columns, got %d: %w", len(row), errs.ErrDbSetup)
		}
		totalSlots, _ := strconv.Atoi(string(row[3]))
		out.Publications[string(row[0])] = PublicationInfo{
			Owner:      string(row[1]),
			Tables:     string(row[2]),
			TotalSlots: totalSlots,
		}
	}

	if len(tables) == 0 {
		return out, nil
	}

	schemas := make([]string, len(tables))
	names := make([]string, len(tables))
	for i, t := range tables {
		schemas[i] = string(t.Schema)
		names[i] = string(t.Name)
	}

	m := pgtype.NewMap()
	schemasArr, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](schemas), nil)
	if err != nil {
		return nil, fmt.Errorf("encode schemas array: %w", err)
	}
	namesArr, err := m.Encode(pgtype.TextArrayOID, pgtype.TextFormatCode, pgtype.FlatArray[string](names), nil)
	if err != nil {
		return nil, fmt.Errorf("encode names array: %w", err)
	}

	queryResult, err := conn.ExecParams(
		ctx,
		replicaIdentitiesSql,
		[]connection.Param{
			{Value: schemasArr, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode},
			{Value: namesArr, OID: pgtype.TextArrayOID, Format: pgtype.TextFormatCode},
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("get replica identities: %w", err)
	}
	for _, vals := range queryResult.Rows {
		if len(vals) < 3 {
			return nil, fmt.Errorf("get replica identities: expected 3 columns, got %d: %w", len(vals), errs.ErrDbSetup)
		}
		key := string(vals[0]) + "." + string(vals[1])
		out.ReplicaIdentities[key] = string(vals[2])
	}

	return out, nil
}

// PrintSummary fetches the catalog summary and logs each section (settings,
// replication slots, publications, replica identities) at info level under the
// "database" log scope.
func PrintSummary(ctx context.Context, conn *connection.Connection, tables []introspector.TableRef) error {
	s, err := GetSummary(ctx, conn, tables)
	if err != nil {
		return err
	}

	// Settings
	settingAttrs := make([]any, 0, len(s.Settings)*2)
	for n, v := range s.Settings {
		settingAttrs = append(settingAttrs, n, v)
	}
	logger.Info("Settings", settingAttrs...)

	// Replication slots
	for n, slot := range s.ReplicationSlots {
		logger.Info("Replication slot", "name", n, "active", slot.Active, "lag", slot.Lag)
	}

	// Publications
	for n, p := range s.Publications {
		logger.Info("Publication", "name", n, "owner", p.Owner, "total_slots", p.TotalSlots, "tables", p.Tables)
	}

	// Replica identities
	riParts := make([]string, 0, len(s.ReplicaIdentities))
	for k, v := range s.ReplicaIdentities {
		riParts = append(riParts, k+"="+v)
	}
	logger.Info("Replica identity", "tables", strings.Join(riParts, ", "))

	return nil
}

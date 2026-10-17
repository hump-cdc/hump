package getsnapshottables

import (
	"context"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
)

// Sink is one sink and the target it renders for a table.
type Sink struct {
	Type   string
	Host   string
	Target string
}

// Table is one configured table as the database describes it, with the sinks
// its snapshot is delivered to.
type Table struct {
	database.TableDescription
	Sinks []Sink
}

// UseCase describes the configured tables as the database holds them.
type UseCase struct {
	cfg config.Configuration
	db  database.Database // db is nil when only the API is described, never served.
}

// NewUseCase returns a use case describing the tables of cfg as db holds them.
func NewUseCase(cfg config.Configuration, db database.Database) *UseCase {
	return &UseCase{
		cfg: cfg,
		db:  db,
	}
}

// Tables describes every configured table found in the database, in
// configuration order, with the filter suggested for its snapshot and its sinks.
func (uc *UseCase) Tables(ctx context.Context) ([]Table, error) {
	if uc.db == nil {
		return []Table{}, nil
	}

	tables := make([]domain.TableIdentifier, 0, len(uc.cfg.Tables))
	configured := make(map[domain.TableIdentifier]config.Table, len(uc.cfg.Tables))
	for _, table := range uc.cfg.Tables {
		id := domain.TableIdentifier{Schema: table.Schema, Name: table.Name}
		tables = append(tables, id)
		configured[id] = table
	}

	descriptions, err := uc.db.GetTables(ctx, tables)
	if err != nil {
		return nil, err
	}

	result := make([]Table, 0, len(descriptions))
	for _, description := range descriptions {
		result = append(result, Table{
			TableDescription: description,
			Sinks:            tableSinks(configured[description.Table]),
		})
	}

	return result, nil
}

// tableSinks renders the sinks of table with the targets they resolve to for it,
// once per distinct sink and target.
func tableSinks(table config.Table) []Sink {
	result := make([]Sink, 0, len(table.Sinks))
	seen := make(map[Sink]struct{}, len(table.Sinks))
	for _, sc := range table.Sinks {
		if sc.Sink == nil {
			continue
		}

		host, target := sc.Sink.Metadata(sink.Context{Schema: table.Schema, Table: table.Name})
		entry := Sink{Type: string(sc.Sink.Type()), Host: host, Target: target}

		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		result = append(result, entry)
	}

	return result
}

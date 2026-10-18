package getconfig

import (
	"slices"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
	"hump/pkg/app/options"
	"hump/pkg/version"
)

// Config is what the web UI knows about the running server.
type Config struct {
	Version    string
	Commit     string
	ConfigPath string
	RawConfig  string
	Database   Database
	Sinks      []Sink // Sinks lists every distinct sink of the configuration once.
	Tables     []Table
}

// Database is the source the server replicates from.
type Database struct {
	Host        string
	Port        uint16
	Name        string
	Publication string
	Slot        string
}

// Sink is one sink endpoint, identified by its type and host.
type Sink struct {
	Type string
	Host string
}

// Target is one table-to-target edge and the operations it receives.
type Target struct {
	Type       string
	Host       string
	Target     string
	Operations []domain.OperationType
}

// Table is one replicated table with its sinks and targets.
type Table struct {
	Schema  domain.TableSchema
	Name    domain.TableName
	Replica string
	Sinks   []Sink
	Targets []Target
}

// UseCase describes the running server from what it was started with.
type UseCase struct {
	opts options.Options
	cfg  config.Configuration
	db   database.Database
}

// NewUseCase returns a use case describing the server started with opts and
// cfg, replicating from db.
func NewUseCase(opts options.Options, cfg config.Configuration, db database.Database) *UseCase {
	return &UseCase{
		opts: opts,
		cfg:  cfg,
		db:   db,
	}
}

// Config describes the server: build, source database, and every table with
// the sinks and targets it is delivered to.
func (uc *UseCase) Config() Config {
	result := Config{
		Version:    version.Version,
		Commit:     version.Commit,
		ConfigPath: uc.opts.ConfigPath,
		RawConfig:  uc.opts.Config,
		Database: Database{
			Host:        uc.db.GetDbHost(),
			Port:        uc.db.GetDbPort(),
			Name:        uc.db.GetDbName(),
			Publication: uc.db.GetPublicationName(),
			Slot:        uc.db.GetSlotName(),
		},
		Sinks:  make([]Sink, 0),
		Tables: make([]Table, 0, len(uc.cfg.Tables)),
	}

	seenAllSinks := map[Sink]struct{}{}

	for _, table := range uc.cfg.Tables {
		sinks := make([]Sink, 0, len(table.Sinks))
		seenSinks := map[Sink]struct{}{}
		for _, sc := range table.Sinks {
			if sc.Sink == nil {
				continue
			}

			// The host is rendered against a zero context: sink URLs are
			// templates, but in practice they resolve from the environment
			// rather than the row being processed. One that does depend on the
			// row renders an empty host here.
			host, _ := sc.Sink.Metadata(sink.Context{})
			entry := Sink{Type: string(sc.Sink.Type()), Host: host}

			if _, ok := seenAllSinks[entry]; !ok {
				seenAllSinks[entry] = struct{}{}
				result.Sinks = append(result.Sinks, entry)
			}

			if _, ok := seenSinks[entry]; ok {
				continue
			}
			seenSinks[entry] = struct{}{}
			sinks = append(sinks, entry)
		}

		result.Tables = append(result.Tables, Table{
			Schema:  table.Schema,
			Name:    table.Name,
			Replica: string(table.Replica),
			Sinks:   sinks,
			Targets: tableTargets(table),
		})
	}

	return result
}

// tableTargets renders the table's edges the way the metrics topology does, so the
// dashboard can match them by type, host and target. Sinks rendering to the same
// edge collapse into one, receiving the union of their operations.
func tableTargets(table config.Table) []Target {
	type key struct{ typ, host, target string }

	result := make([]Target, 0, len(table.Sinks))
	idx := map[key]int{}
	for _, sc := range table.Sinks {
		if sc.Sink == nil {
			continue
		}

		host, target := sc.Sink.Metadata(sink.Context{Schema: table.Schema, Table: table.Name})
		k := key{string(sc.Sink.Type()), host, target}

		i, ok := idx[k]
		if !ok {
			i = len(result)
			idx[k] = i
			result = append(result, Target{Type: k.typ, Host: host, Target: target})
		}

		result[i].Operations = union(result[i].Operations, sc.Operations)
	}

	return result
}

// union merges b into a, keeping the canonical operation order.
func union(a, b []domain.OperationType) []domain.OperationType {
	out := make([]domain.OperationType, 0, len(domain.OperationTypes))
	for _, op := range domain.OperationTypes {
		if slices.Contains(a, op) || slices.Contains(b, op) {
			out = append(out, op)
		}
	}

	return out
}

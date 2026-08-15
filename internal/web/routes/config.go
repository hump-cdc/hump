package routes

import (
	"context"
	"hump/config"
	"hump/internal/adapter/postgres"
	"hump/internal/domain"
	"hump/pkg/app/options"
	"hump/pkg/version"
	"net/http"

	"hump/internal/port/sink"

	"github.com/danielgtaylor/huma/v2"
)

// ConfigSink identifies one sink endpoint: its type together with the connection
// host its URL renders to. The pair is what distinguishes two sinks of the same
// type pointed at different clusters, and it matches the `type`/`host` labels the
// telemetry exporter attaches to per-sink metrics.
type ConfigSink struct {
	Type string `json:"type" doc:"Sink type"`
	Host string `json:"host" doc:"Connection host the sink URL renders to"`
}

type ConfigTable struct {
	Schema  domain.TableSchema `json:"schema"  doc:"Table schema"`
	Name    domain.TableName   `json:"name"    doc:"Table name"`
	Replica string             `json:"replica" doc:"Replica identity managed by hump"`
	Sinks   []ConfigSink       `json:"sinks"   doc:"Sinks connected to the table"`
}

type ConfigResponse struct {
	Body struct {
		App struct {
			Version string `json:"version"          doc:"hump build version"`
			Commit  string `json:"commit,omitempty" doc:"hump build commit hash"`
		} `json:"app" doc:"Application info"`
		ConfigPath string `json:"configPath,omitempty" doc:"Config file path when loaded from a file"`
		RawConfig  string `json:"rawConfig,omitempty"  doc:"Raw configuration provided at startup"`
		Database   struct {
			Host string `json:"host" doc:"PostgreSQL host"`
			Port uint16 `json:"port" doc:"PostgreSQL port"`
			Name string `json:"name" doc:"PostgreSQL database name"`

			PublicationName     string `json:"publication" doc:"PostgreSQL publication name"`
			ReplicationSlotName string `json:"slot"        doc:"PostgreSQL replication slot name"`
		} `json:"database" doc:"PostgreSQL database configuration"`

		Sinks  []ConfigSink  `json:"sinks"  doc:"Sinks present in the configuration"`
		Tables []ConfigTable `json:"tables" doc:"Replicated tables"`
	}
}

func RegisterConfig(api huma.API, opts options.Options, cfg config.Configuration, db *postgres.Postgres) {
	huma.Register(api, huma.Operation{
		OperationID: "config.get",
		Method:      http.MethodGet,
		Path:        "/api/config",
		Summary:     "Get server configuration",
		Tags:        []string{"Config"},
	}, handler(opts, cfg, db))
}

func handler(opts options.Options, cfg config.Configuration, db *postgres.Postgres) func(ctx context.Context, _ *struct{}) (*ConfigResponse, error) {
	return func(_ context.Context, _ *struct{}) (*ConfigResponse, error) {
		resp := ConfigResponse{}
		resp.Body.App.Version = version.Version
		resp.Body.App.Commit = version.Commit
		resp.Body.ConfigPath = opts.ConfigPath
		resp.Body.RawConfig = opts.Config
		resp.Body.Database.Host = db.GetDbHost()
		resp.Body.Database.Port = db.GetDbPort()
		resp.Body.Database.Name = db.GetDbName()
		resp.Body.Database.PublicationName = db.GetPublicationName()
		resp.Body.Database.ReplicationSlotName = db.GetSlotName()

		resp.Body.Sinks = make([]ConfigSink, 0)
		seenAllSinks := map[ConfigSink]struct{}{}

		resp.Body.Tables = make([]ConfigTable, 0, len(cfg.Tables))

		for _, table := range cfg.Tables {
			sinks := make([]ConfigSink, 0, len(table.Sinks))
			seenSinks := map[ConfigSink]struct{}{}
			for _, sc := range table.Sinks {
				if sc.Sink == nil {
					continue
				}

				// The host is rendered against a zero context: sink URLs are
				// templates, but in practice they resolve from the environment
				// rather than the row being processed. One that does depend on the
				// row renders an empty host here.
				host, _ := sc.Sink.Metadata(sink.SinkContext{})
				entry := ConfigSink{Type: string(sc.Sink.Type()), Host: host}

				if _, ok := seenAllSinks[entry]; !ok {
					seenAllSinks[entry] = struct{}{}
					resp.Body.Sinks = append(resp.Body.Sinks, entry)
				}

				if _, ok := seenSinks[entry]; ok {
					continue
				}
				seenSinks[entry] = struct{}{}
				sinks = append(sinks, entry)
			}

			resp.Body.Tables = append(resp.Body.Tables, ConfigTable{
				Schema:  table.Schema,
				Name:    table.Name,
				Replica: string(table.Replica),
				Sinks:   sinks,
			})
		}

		return &resp, nil
	}
}

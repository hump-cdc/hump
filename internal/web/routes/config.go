package routes

import (
	"context"
	"hump/internal/source/postgres"
	"hump/internal/source/postgres/connection"
	"hump/internal/types/config"
	"hump/pkg/app/options"
	"hump/pkg/version"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"hump/internal/port/sink"
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
	Schema  string       `json:"schema"  doc:"Table schema"`
	Name    string       `json:"name"    doc:"Table name"`
	Replica string       `json:"replica" doc:"Replica identity managed by hump"`
	Sinks   []ConfigSink `json:"sinks"   doc:"Sinks connected to the table"`
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

			MaxReplicationSlots int `json:"maxReplicationSlots" doc:"PostgreSQL max_replication_slots setting"`
			MaxWalSenders       int `json:"maxWalSenders"       doc:"PostgreSQL max_wal_senders setting"`
		} `json:"database" doc:"PostgreSQL database configuration"`

		Sinks  []ConfigSink  `json:"sinks"  doc:"Sinks present in the configuration"`
		Tables []ConfigTable `json:"tables" doc:"Replicated tables"`
	}
}

func RegisterConfig(api huma.API, opts options.Options, cfg config.Configuration) {
	huma.Register(api, huma.Operation{
		OperationID: "config.get",
		Method:      http.MethodGet,
		Path:        "/api/config",
		Summary:     "Get server configuration",
		Tags:        []string{"Config"},
	}, handler(opts, cfg))
}

func handler(opts options.Options, cfg config.Configuration) func(ctx context.Context, _ *struct{}) (*ConfigResponse, error) {
	return func(ctx context.Context, _ *struct{}) (*ConfigResponse, error) {
		endpoint, err := connection.Describe(opts.DatabaseUrl)
		if err != nil {
			return nil, err
		}

		conn, err := connection.New(opts.DatabaseUrl).CreateConnection(ctx)
		if err != nil {
			return nil, err
		}
		defer conn.Close(ctx)

		dbConfig, err := postgres.GetConfig(ctx, conn)
		if err != nil {
			return nil, err
		}

		resp := ConfigResponse{}
		resp.Body.App.Version = version.Version
		resp.Body.App.Commit = version.Commit
		resp.Body.ConfigPath = opts.ConfigPath
		resp.Body.RawConfig = opts.Config
		resp.Body.Database.Host = endpoint.Host
		resp.Body.Database.Port = endpoint.Port
		resp.Body.Database.Name = string(endpoint.Database)
		resp.Body.Database.PublicationName = opts.PublicationName
		resp.Body.Database.ReplicationSlotName = opts.ReplicationSlotName
		resp.Body.Database.MaxReplicationSlots = dbConfig.MaxReplicationSlots
		resp.Body.Database.MaxWalSenders = dbConfig.MaxWalSenders

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

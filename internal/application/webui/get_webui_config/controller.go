package getwebuiconfig

import (
	"context"
	"hump/config"
	"hump/internal/port/database"
	"hump/internal/port/sink"
	"hump/pkg/app/options"
	"hump/pkg/version"
)

type Controller struct {
	opts options.Options
	cfg  config.Configuration
	db   database.Database
}

func NewController(opts options.Options, cfg config.Configuration, db database.Database) *Controller {
	return &Controller{
		opts: opts,
		cfg:  cfg,
		db:   db,
	}
}

func (c *Controller) Handler() func(ctx context.Context, _ *struct{}) (*ConfigResponse, error) {
	return func(_ context.Context, _ *struct{}) (*ConfigResponse, error) {
		resp := ConfigResponse{}
		resp.Body.App.Version = version.Version
		resp.Body.App.Commit = version.Commit
		resp.Body.ConfigPath = c.opts.ConfigPath
		resp.Body.RawConfig = c.opts.Config
		resp.Body.Database.Host = c.db.GetDbHost()
		resp.Body.Database.Port = c.db.GetDbPort()
		resp.Body.Database.Name = c.db.GetDbName()
		resp.Body.Database.PublicationName = c.db.GetPublicationName()
		resp.Body.Database.ReplicationSlotName = c.db.GetSlotName()

		resp.Body.Sinks = make([]ConfigSink, 0)
		seenAllSinks := map[ConfigSink]struct{}{}

		resp.Body.Tables = make([]ConfigTable, 0, len(c.cfg.Tables))

		for _, table := range c.cfg.Tables {
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

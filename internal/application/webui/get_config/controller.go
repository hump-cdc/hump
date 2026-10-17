// Package getconfig serves the config of the web UI: the build, the startup
// options, the source database and every configured table with its sinks.
package getconfig

import (
	"context"
)

// Controller serves GET /api/config.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller serving the description useCase builds.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Handler returns the huma handler rendering the server description.
func (c *Controller) Handler() func(ctx context.Context, _ *struct{}) (*ConfigResponse, error) {
	return func(_ context.Context, _ *struct{}) (*ConfigResponse, error) {
		cfg := c.useCase.Config()

		resp := ConfigResponse{}
		resp.Body.App.Version = cfg.Version
		resp.Body.App.Commit = cfg.Commit
		resp.Body.ConfigPath = cfg.ConfigPath
		resp.Body.RawConfig = cfg.RawConfig
		resp.Body.Database.Host = cfg.Database.Host
		resp.Body.Database.Port = cfg.Database.Port
		resp.Body.Database.Name = cfg.Database.Name
		resp.Body.Database.PublicationName = cfg.Database.Publication
		resp.Body.Database.ReplicationSlotName = cfg.Database.Slot
		resp.Body.Sinks = configSinks(cfg.Sinks)

		resp.Body.Tables = make([]ConfigTable, 0, len(cfg.Tables))
		for _, table := range cfg.Tables {
			resp.Body.Tables = append(resp.Body.Tables, ConfigTable{
				Schema:  table.Schema,
				Name:    table.Name,
				Replica: table.Replica,
				Sinks:   configSinks(table.Sinks),
				Targets: configTargets(table.Targets),
			})
		}

		return &resp, nil
	}
}

// configSinks renders sink endpoints.
func configSinks(source []Sink) []ConfigSink {
	result := make([]ConfigSink, 0, len(source))
	for _, s := range source {
		result = append(result, ConfigSink(s))
	}

	return result
}

// configTargets renders the table-to-target edges of a table.
func configTargets(source []Target) []ConfigTarget {
	result := make([]ConfigTarget, 0, len(source))
	for _, t := range source {
		result = append(result, ConfigTarget(t))
	}

	return result
}

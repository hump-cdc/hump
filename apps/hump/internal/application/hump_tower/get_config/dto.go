package getconfig

import "hump/internal/domain"

// ConfigSink is a sink endpoint a table is connected to.
type ConfigSink struct {
	Type string `json:"type" doc:"Sink type"`
	Host string `json:"host" doc:"Connection host the sink URL renders to"`
}

// ConfigTarget is a table-to-target edge with the operations it carries.
type ConfigTarget struct {
	Type       string                 `json:"type"       doc:"Sink type"`
	Host       string                 `json:"host"       doc:"Connection host the sink URL renders to"`
	Target     string                 `json:"target"     doc:"Target rendered for the table, as the metrics topology reports it"`
	Operations []domain.OperationType `json:"operations" doc:"Row operations delivered to the target" enum:"insert,update,delete,truncate"`
}

// ConfigTable is a configured table with its sinks and targets.
type ConfigTable struct {
	Schema  domain.TableSchema `json:"schema"  doc:"Table schema"`
	Name    domain.TableName   `json:"name"    doc:"Table name"`
	Replica string             `json:"replica" doc:"Replica identity managed by hump"`
	Sinks   []ConfigSink       `json:"sinks"   doc:"Sinks connected to the table"`
	Targets []ConfigTarget     `json:"targets" doc:"Table-to-target edges, with the operations each receives"`
}

// ConfigResponse is the body of GET /api/config.
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

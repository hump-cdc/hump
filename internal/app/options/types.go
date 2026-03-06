package options

import "errors"

var (
	ErrConfigConflict = errors.New("--config and --config-file are mutually exclusive, please provide only one of them")
	ErrConfigMissing  = errors.New("one of --config / --config-file is required")
	ErrHelp           = errors.New("help requested")
)

type LogFormat string

const (
	LogFormatPretty LogFormat = "pretty"
	LogFormatJSON   LogFormat = "json"
)

type Options struct {
	DatabaseUrl         string `short:"d" long:"database-url" env:"DATABASE_URL" description:"Database connection URL"`
	Config              string `short:"c" long:"config" description:"Table configuration [$CONFG]"`
	ConfigPath          string `short:"f" long:"config-file" description:"Path to table configuration file [$CONFIG_FILE]"`
	PublicationName     string `short:"p" long:"publication" env:"PUB" description:"PostgreSQL publication name" default:"hump_pub"`
	ReplicationSlotName string `short:"s" long:"slot" env:"SLOT" description:"PostgreSQL replication slot name" default:"hump_slot"`
	TelemetryPort       *uint  `short:"t" long:"telemetry-port" env:"TELEMETRY_PORT" description:"Port for telemetry endpoints"`
}

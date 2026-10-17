package options

import "errors"

var (
	// ErrConfigConflict is returned by Parse when both --config and --config-file are set.
	ErrConfigConflict = errors.New("--config and --config-file are mutually exclusive, please provide only one of them")
	// ErrConfigMissing is returned by Parse when no source yields a configuration.
	ErrConfigMissing = errors.New("one of --config / --config-file is required")
	// ErrHelp is returned by Parse when -h/--help is requested.
	ErrHelp = errors.New("help requested")
)

// Options is the parsed command line, with the configuration source resolved
// into Config.
type Options struct {
	DatabaseUrl         string `short:"d" long:"database-url" env:"DATABASE_URL" description:"Database connection URL"`
	Config              string `short:"c" long:"config" description:"Table configuration [$CONFG]"`
	ConfigPath          string `short:"f" long:"config-file" description:"Path to table configuration file [$CONFIG_FILE]"`
	PublicationName     string `short:"p" long:"publication" env:"PUB" description:"PostgreSQL publication name" default:"hump_pub"`
	ReplicationSlotName string `short:"s" long:"slot" env:"SLOT" description:"PostgreSQL replication slot name" default:"hump_slot"`
	TelemetryPort       *uint  `short:"t" long:"telemetry-port" env:"TELEMETRY_PORT" description:"Port for telemetry endpoints"`
	WebPort             *uint  `short:"w" long:"web-port" env:"WEB_PORT" description:"Port for web ui"`
	Cleanup             bool   `long:"cleanup" description:"Run cleanup: drop the publication and replication slot, then exit"`
}

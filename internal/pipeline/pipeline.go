package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	cfg "hump/internal/config"
	"hump/internal/log"
	"hump/internal/processor"
	"hump/internal/source/postgres"
	"hump/internal/source/postgres/connection"
	"hump/internal/source/postgres/wal"
	"hump/internal/telemetry/health"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/config"
	"hump/internal/types/errs"

	"github.com/goccy/go-yaml"
)

var logger = log.Scope("pipeline")

// Pipeline orchestrates change-data-capture from a Postgres logical replication
// slot to the configured downstream sinks.
type Pipeline interface {
	// Start parses the configuration, connects to Postgres using databaseUrl,
	// and begins streaming WAL changes for the given publication and slot until
	// ctx is cancelled or an unrecoverable error occurs.
	Start(ctx context.Context, databaseUrl, config, publicationName, slotName string) error
}

type pipeline struct {
	health  *health.Health
	metrics *metrics.Metrics
}

func New(health *health.Health, metrics *metrics.Metrics) Pipeline {
	return &pipeline{
		health:  health,
		metrics: metrics,
	}
}

// Start runs the pipeline.
func (p *pipeline) Start(ctx context.Context, databaseUrl, config, publicationName, slotName string) error {
	cfg, err := cfg.Parse(config)

	if err != nil {
		logger.Error(err.Error())
		return errs.ErrConfigurationParse
	}

	cfgResult, _ := yaml.Marshal(&cfg)
	logger.Debug(fmt.Sprintf("Configuration:\n%s\n", string(cfgResult)))

	for _, t := range cfg.Tables {
		logger.Debug("Configuration",
			"schema", t.Schema,
			"table", t.Name,
			"sinks", formatSinks(t.Sinks),
		)
	}

	connector := connection.New(databaseUrl)

	registry, introspector, err := postgres.Configure(ctx, connector, cfg, publicationName, slotName)
	if err != nil {
		return fmt.Errorf("unable to configure to database: %w", err)
	}

	conn, err := connector.CreateConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	p.health.Postgres.Update(true, nil)

	err = postgres.PrintSummary(ctx, conn, registry.List())
	if err != nil {
		return fmt.Errorf("failed to get database summary: %w", err)
	}

	proc := processor.New(introspector, p.metrics)

	w, err := wal.New(ctx, connector, publicationName, slotName, p.health, p.metrics, proc.Process)
	if err != nil {
		return fmt.Errorf("failed to initialize WAL processing: %w", err)
	}

	err = w.Start(ctx)
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	defer w.Close(closeCtx)

	return err
}

func formatSinks(sinks []config.SinkConfig) string {
	parts := make([]string, 0, len(sinks))
	for _, s := range sinks {
		parts = append(parts, s.Sink.DisplayName())
	}
	return strings.Join(parts, ", ")
}

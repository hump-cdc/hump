package pipeline

import (
	"context"
	"fmt"
	"time"

	"hump/internal/application/process_operation"
	"hump/internal/port/sink"
	"hump/internal/source/postgres"
	"hump/internal/source/postgres/connection"
	"hump/internal/source/postgres/wal"
	"hump/internal/telemetry/health"
	"hump/internal/telemetry/metrics"
	"hump/internal/types/config"
	"hump/internal/types/stream"
	"hump/pkg/telemetry/log"
)

var logger = log.Scope("pipeline")

// Pipeline orchestrates change-data-capture from a Postgres logical replication
// slot to the configured downstream sinks.
type Pipeline interface {
	// Start parses the configuration, connects to Postgres using databaseUrl,
	// and begins streaming WAL changes for the given publication and slot until
	// ctx is cancelled or an unrecoverable error occurs.
	Start(ctx context.Context, databaseUrl string, cfg config.Configuration, publicationName string, slotName string) error
}

type pipeline struct {
	health    *health.Health
	publisher stream.Publisher
	metrics   *metrics.Metrics
	sinks     sink.Sink
}

func New(health *health.Health, publisher stream.Publisher, metrics *metrics.Metrics, sinks sink.Sink) Pipeline {
	return &pipeline{
		health:    health,
		publisher: publisher,
		metrics:   metrics,
		sinks:     sinks,
	}
}

// Start runs the pipeline.
func (p *pipeline) Start(ctx context.Context, databaseUrl string, cfg config.Configuration, publicationName string, slotName string) error {

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

	proc := process_operation.NewUseCase(introspector, p.sinks, p.publisher, p.metrics)

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

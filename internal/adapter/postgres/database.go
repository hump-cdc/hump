package postgres

import (
	"context"
	"fmt"
	"hump/config"
	"hump/internal/adapter/postgres/connection"
	"hump/internal/adapter/postgres/wal"
	"hump/internal/port/database"
	"time"
)

type Database struct {
	databaseUrl     string
	cfg             config.Configuration
	publicationName string
	slotName        string
}

func NewDatabase(databaseUrl string, cfg config.Configuration, publicationName string, slotName string) database.Database {
	return &Database{
		databaseUrl:     databaseUrl,
		cfg:             cfg,
		publicationName: publicationName,
		slotName:        slotName,
	}
}

func (d *Database) GetDbName() string {
	return ""
}

func (d *Database) GetPublicationName() string {
	return d.publicationName
}

func (d *Database) GetSlotName() string {
	return d.slotName
}

func (d *Database) Listen(ctx context.Context, handler database.ProcessMessageHandler) error {
	connector := connection.New(d.databaseUrl)

	_, introspector, err := Configure(ctx, connector, d.cfg, d.publicationName, d.slotName)
	if err != nil {
		return fmt.Errorf("unable to configure to database: %w", err)
	}

	conn, err := connector.CreateConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// TODO: p.health.Postgres.Update(true, nil)

	// err = PrintSummary(ctx, conn, registry.List())
	// if err != nil {
	// 	return fmt.Errorf("failed to get database summary: %w", err)
	// }

	// proc := process_operation.NewUseCase(introspector, p.sinks, p.publisher, p.metrics)

	w, err := wal.New(ctx, connector, d.publicationName, d.slotName, introspector)
	if err != nil {
		return fmt.Errorf("failed to initialize WAL processing: %w", err)
	}

	err = w.Start(ctx, handler)
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	defer w.Stop(closeCtx)

	return err
}

func (d *Database) Cleanup(ctx context.Context) error {
	connector := connection.New(d.databaseUrl)
	return Cleanup(ctx, connector, d.publicationName, d.slotName)
}

package test

import (
	"context"
	"errors"
	"fmt"
	"hump/config"
	"hump/internal/pipeline"
	"hump/internal/telemetry/health"
	"hump/internal/telemetry/metrics"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

// setupOptions holds the configurable inputs for setup. Zero values are
// replaced with defaults in setup before the options are applied.
type setupOptions struct {
	config      string
	pubName     string
	slotName    string
	databaseURL string
}

// setupOption configures setupOptions.
type setupOption func(*setupOptions)

// withConfig sets the pipeline YAML configuration.
func withConfig(c string) setupOption { return func(o *setupOptions) { o.config = c } }

// withPubName sets the logical replication publication name.
func withPubName(n string) setupOption { return func(o *setupOptions) { o.pubName = n } }

// withSlotName sets the logical replication slot name.
func withSlotName(n string) setupOption { return func(o *setupOptions) { o.slotName = n } }

// withDatabaseURL sets the Postgres connection string.
func withDatabaseURL(u string) setupOption { return func(o *setupOptions) { o.databaseURL = u } }

// setup starts the pipeline, waits for the WAL sender to become ready, and
// connects to Postgres. It returns the database connection, a teardown func
// that cancels the context and waits for the pipeline to exit before closing
// the connection, a cancel func that only cancels the pipeline context, and an
// error. Kafka is connected by the caller.
func setup(t *testing.T, opts ...setupOption) (context.Context, *pgx.Conn, context.CancelFunc, func() error, error) {
	t.Helper()

	o := setupOptions{
		pubName:  fmt.Sprintf("test_pub_%d", time.Now().UnixNano()),
		slotName: fmt.Sprintf("test_slot_%d", time.Now().UnixNano()),
	}
	for _, opt := range opts {
		opt(&o)
	}

	testCtx, teardown := context.WithCancel(t.Context())
	g, ctx := errgroup.WithContext(testCtx)
	wait := func() error {
		if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}

	t.Log("Connecting to postgres...")
	db, err := connectPostgres(ctx, o.databaseURL)
	if err != nil {
		teardown()
		return ctx, nil, nil, wait, fmt.Errorf("connect postgres: %w", err)
	}
	t.Log("Connected to postgres")

	h, _, _, _ := health.New()
	cfg, err := config.Parse(o.config)
	if err != nil {
		teardown()
		return ctx, nil, nil, wait, fmt.Errorf("parse config: %w", err)
	}

	g.Go(func() error {
		t.Log("Starting pipeline...")
		m, _ := metrics.New("hump")
		p := pipeline.New(h, nil, m)
		return p.Start(ctx, o.databaseURL, cfg, o.pubName, o.slotName)
	})

	// Wait for the WAL sender to come up before producing messages, otherwise a
	// write can race ahead of logical replication and be missed.
	if err := waitWalReady(ctx, h, 10*time.Second); err != nil {
		teardown()
		return ctx, nil, nil, wait, fmt.Errorf("pipeline did not start WAL processing: %w", err)
	}
	t.Log("Pipeline WAL sender is ready")

	return ctx, db, teardown, wait, nil
}

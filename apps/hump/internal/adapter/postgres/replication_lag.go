package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"
)

//go:embed sql/replication_lag.sql
var replicationLagSql string

// replicationLagInterval is the wait between replication lag polls. It matches
// standbyTimeout, the cadence at which confirmed_flush_lsn can move at all.
const replicationLagInterval = 10 * time.Second

// replicationLagPoller periodically reads how much WAL the replication slot
// retains and reports it as the replication lag metric.
type replicationLagPoller struct {
	connection *connection // connection must not be shared: pgconn is not safe for concurrent use.
	slotName   string
	interval   time.Duration
	metrics    *metrics
}

// newReplicationLagPoller returns a poller for slotName on conn, which it
// borrows: the caller keeps it open until run returns.
func newReplicationLagPoller(conn *connection, slotName string, m *metrics) *replicationLagPoller {
	return &replicationLagPoller{
		connection: conn,
		slotName:   slotName,
		interval:   replicationLagInterval,
		metrics:    m,
	}
}

// run polls immediately and then every interval until ctx is cancelled. A
// failed poll is logged and leaves the last reported value in place.
func (p *replicationLagPoller) run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		if err := p.poll(ctx); err != nil && ctx.Err() == nil {
			logger.Warn("Unable to read replication lag", "slot", p.slotName, "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// poll reads the slot's retained WAL and sets the metric. A missing slot counts
// as no lag; a NULL confirmed_flush_lsn is an error, as there is nothing to measure.
func (p *replicationLagPoller) poll(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, p.interval)
	defer cancel()

	result, err := p.connection.ExecParams(ctx, replicationLagSql, []param{textParam(p.slotName)}, nil)
	if err != nil {
		return fmt.Errorf("query replication lag: %w", err)
	}

	if len(result.Rows) == 0 {
		p.metrics.setReplicationLag(0)
		return nil
	}

	value := result.Rows[0]
	if len(value) == 0 || value[0] == nil {
		return fmt.Errorf("replication slot %q has no confirmed flush position", p.slotName)
	}

	lag, err := strconv.ParseFloat(string(value[0]), 64)
	if err != nil {
		return fmt.Errorf("parse replication lag %q: %w", value[0], err)
	}

	p.metrics.setReplicationLag(lag)

	return nil
}

package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"hump/internal/domain"
)

// param is a single bind parameter for ExecParams.
type param struct {
	// Value is the wire-format encoded value.
	Value []byte
	// OID is the PostgreSQL type OID; 0 lets the server infer it.
	OID uint32
	// Format is 0 for text, 1 for binary.
	Format int16
}

// connection guards a single *pgconn.PgConn and reconnects when it is found
// closed after a failed call.
type connection struct {
	connectionConfig *pgconn.Config
	dbHost           string
	dbPort           uint16
	dbName           domain.DbName

	connection *pgconn.PgConn
	mu         sync.Mutex

	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

// newConnection returns a connection for connectionString, opening the socket
// lazily on first use. It only fails on an unusable connection string; the
// caller must Close the result.
func newConnection(connectionString string, opts ...option) (*connection, error) {
	options := newConnectionOptions(connectionString, opts...)

	connectionConfig, dbHost, dbPort, dbName, err := options.GetConnectionConfig()
	if err != nil {
		return nil, err
	}

	connectionConfig.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		logger.Info("NOTICE: "+n.Message, slog.String("severity", n.Severity))
	}

	return &connection{
		connectionConfig: connectionConfig,
		dbHost:           dbHost,
		dbPort:           dbPort,
		dbName:           dbName,
		maxAttempts:      options.maxAttempts,
		baseDelay:        options.baseDelay,
		maxDelay:         options.maxDelay,
	}, nil
}

// GetDbName returns the database name parsed from the connection string.
func (c *connection) GetDbName() domain.DbName {
	return c.dbName
}

// GetDbHost returns the host parsed from the connection string.
func (c *connection) GetDbHost() string {
	return c.dbHost
}

// GetDbPort returns the port parsed from the connection string.
func (c *connection) GetDbPort() uint16 {
	return c.dbPort
}

// Raw returns the underlying pgconn session, opening or reopening it as needed.
// The handle escapes the mutex, so a concurrent Close or reconnect invalidates
// it and callers must re-acquire.
func (c *connection) Raw(ctx context.Context) (*pgconn.PgConn, error) {
	return c.acquire(ctx)
}

// Close terminates the underlying session; safe when none was opened, it is
// already closed, or Close is called twice. Not a terminal state: a later
// Exec, ExecParams or Raw opens a new session, so a shutdown stops callers first.
func (c *connection) Close(ctx context.Context) error {
	c.mu.Lock()
	connection := c.connection
	c.connection = nil
	c.mu.Unlock()

	if connection == nil {
		return nil
	}

	return connection.Close(ctx)
}

// Exec runs sql and materializes all results. sql may run more than once, so it
// must be idempotent; on error the results are those of the last attempt that
// reached the server and are nil or partial.
func (c *connection) Exec(ctx context.Context, sql string) ([]*pgconn.Result, error) {
	var results []*pgconn.Result

	err := c.run(ctx, func(connection *pgconn.PgConn) error {
		var execErr error
		results, execErr = connection.Exec(ctx, sql).ReadAll()
		return execErr
	})

	return results, err
}

// ExecParams runs a single parameterized statement and materializes the result.
// sql may run more than once, so it must be idempotent; on error the result is
// the last attempt that reached the server, or nil if none did.
func (c *connection) ExecParams(
	ctx context.Context,
	sql string,
	params []param,
	resultFormats []int16,
) (*pgconn.Result, error) {
	values := make([][]byte, len(params))
	oids := make([]uint32, len(params))
	formats := make([]int16, len(params))
	for i, p := range params {
		values[i] = p.Value
		oids[i] = p.OID
		formats[i] = p.Format
	}

	var result *pgconn.Result

	err := c.run(ctx, func(connection *pgconn.PgConn) error {
		result = connection.ExecParams(ctx, sql, values, oids, formats, resultFormats).Read()
		return result.Err
	})

	return result, err
}

// run calls fn on an acquired session, retrying with backoff up to c.maxAttempts
// times when acquire fails or the session dies mid-call. Other errors are
// returned as is.
func (c *connection) run(ctx context.Context, fn func(*pgconn.PgConn) error) error {
	var lastErr error

	for attempt := range c.maxAttempts {
		connection, acquireErr := c.acquire(ctx)

		if acquireErr != nil {
			lastErr = acquireErr

			if attempt == c.maxAttempts-1 {
				break
			}

			if backoffErr := c.backoff(ctx, attempt); backoffErr != nil {
				return backoffErr
			}
			continue
		}

		if err := fn(connection); err != nil {
			lastErr = err

			// Retry only when the session died mid-call and ctx is still live: on a
			// cancelled ctx the redial is doomed and would mask err with ctx.Err().
			if !connection.IsClosed() || ctx.Err() != nil {
				return err
			}

			if attempt == c.maxAttempts-1 {
				break
			}

			if backoffErr := c.backoff(ctx, attempt); backoffErr != nil {
				return backoffErr
			}

			continue
		}
		return nil
	}

	return fmt.Errorf("exec failed after %d attempts: %w", c.maxAttempts, lastErr)
}

// acquire returns the cached session, opening one when it is missing or closed.
// The dial holds c.mu, so concurrent callers queue up and share one session.
func (c *connection) acquire(ctx context.Context) (*pgconn.PgConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connection != nil && !c.connection.IsClosed() {
		return c.connection, nil
	}

	// Queueing on the mutex can outlast the caller, and a mutex cannot be waited
	// with a context, so check it here rather than dialing for a caller that left.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Drop the dead handle first so a failed dial does not leave it cached.
	c.connection = nil

	connection, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}

	c.connection = connection

	return connection, nil
}

// connect opens a pgconn session with retries and exponential backoff. It is
// called with c.mu held, so it neither touches c.connection nor retakes the lock.
func (c *connection) connect(ctx context.Context) (*pgconn.PgConn, error) {
	var lastErr error

	for attempt := range c.maxAttempts {
		connection, err := pgconn.ConnectConfig(ctx, c.connectionConfig)
		if err == nil {
			return connection, nil
		}

		lastErr = err
		if attempt == c.maxAttempts-1 {
			break
		}

		if waitErr := c.backoff(ctx, attempt); waitErr != nil {
			return nil, waitErr
		}
	}

	return nil, fmt.Errorf("connect failed after %d attempts: %w", c.maxAttempts, lastErr)
}

// backoff waits c.baseDelay<<attempt, clamped to c.maxDelay, or returns
// ctx.Err() if ctx is cancelled first.
func (c *connection) backoff(ctx context.Context, attempt int) error {
	delay := c.baseDelay << attempt
	if delay <= 0 || delay > c.maxDelay {
		delay = c.maxDelay
	}

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Package connection wraps pgx's low-level pgconn to provide a self-healing
// PostgreSQL session: Connection retries on connection-level errors and
// transparently reopens the underlying socket when it is detected as closed.
// Connector is the entry point for opening regular or replication sessions.
package connection

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"
)

// Param is a single bind parameter for ExecParams.
type Param struct {
	// Value is the wire-format encoded parameter value.
	Value []byte
	// OID is the PostgreSQL type OID; 0 lets the server infer the type.
	OID uint32
	// Format is 0 for text or 1 for binary encoding of Value.
	Format int16
}

// Connection guards a single *pgconn.PgConn and reconnects when the
// underlying connection is detected as closed after a failed call.
type Connection struct {
	connectionString string
	connection       *pgconn.PgConn
	mu               sync.Mutex
}

// Close terminates the underlying pgconn session. It is safe to call on a
// Connection whose session was never opened or has already been closed.
func (c *Connection) Close(ctx context.Context) error {
	c.mu.Lock()
	connection := c.connection
	c.connection = nil
	c.mu.Unlock()

	if connection == nil {
		return nil
	}

	return connection.Close(ctx)
}

// Exec runs sql and materializes all results.
func (c *Connection) Exec(ctx context.Context, sql string) ([]*pgconn.Result, error) {
	var results []*pgconn.Result

	err := c.run(ctx, func(connection *pgconn.PgConn) error {
		var execErr error
		results, execErr = connection.Exec(ctx, sql).ReadAll()
		return execErr
	})

	return results, err
}

// ExecParams runs a single parameterized statement and materializes the result.
func (c *Connection) ExecParams(
	ctx context.Context,
	sql string,
	params []Param,
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

// run invokes fn against an acquired session, retrying up to
// connectionMaxAttempts times when the session is closed mid-call. Non
// connection-level errors are returned to the caller without retry.
func (c *Connection) run(ctx context.Context, fn func(*pgconn.PgConn) error) error {
	var lastErr error

	for attempt := range connectionMaxAttempts {
		connection, err := c.acquire(ctx)

		if err != nil {
			lastErr = err

			if waitErr := backoff(ctx, attempt); waitErr != nil {
				return waitErr
			}
			continue
		}

		if err := fn(connection); err != nil {
			lastErr = err

			if !connection.IsClosed() {
				return err
			}

			continue
		}
		return nil
	}

	return fmt.Errorf("exec failed after %d attempts: %w", connectionMaxAttempts, lastErr)
}

// acquire returns the cached session or opens a new one when the previous
// session is missing or closed. Concurrent racers reuse the first winner.
func (c *Connection) acquire(ctx context.Context) (*pgconn.PgConn, error) {
	c.mu.Lock()
	connection := c.connection
	c.mu.Unlock()

	if connection != nil && !connection.IsClosed() {
		return connection, nil
	}

	newConnection, _, err := connect(ctx, c.connectionString)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if c.connection != nil && !c.connection.IsClosed() {
		existing := c.connection
		c.mu.Unlock()
		_ = newConnection.Close(ctx)
		return existing, nil
	}
	c.connection = newConnection
	c.mu.Unlock()

	return newConnection, nil
}

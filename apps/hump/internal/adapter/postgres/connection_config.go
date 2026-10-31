package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"hump/internal/domain"
)

// connectionOptions holds the connection string and the settings applied by Options.
type connectionOptions struct {
	connectionString string
	replication      bool
	maxAttempts      int
	baseDelay        time.Duration
	maxDelay         time.Duration
}

// option customizes a connection created by newConnection.
type option func(c *connectionOptions)

// newConnectionOptions builds the options for connectionString from opts, starting
// from the default retry policy: five attempts, doubling from 100ms, capped at 5s.
func newConnectionOptions(connectionString string, opts ...option) *connectionOptions {
	c := &connectionOptions{
		connectionString: connectionString,
		maxAttempts:      5,
		baseDelay:        100 * time.Millisecond,
		maxDelay:         5 * time.Second,
	}
	c.apply(opts...)

	return c
}

// GetConnectionConfig parses the connection string, applies the settings from
// Options, and returns the pgconn config plus the host, port and database name
// it resolves to. The config carries the credentials.
func (c *connectionOptions) GetConnectionConfig() (*pgconn.Config, string, uint16, domain.DbName, error) {
	cfg, err := pgconn.ParseConfig(c.connectionString)
	if err != nil {
		return nil, "", 0, "", err
	}

	if c.replication {
		cfg.RuntimeParams["replication"] = "database"
	}
	for name := range cfg.RuntimeParams {
		if isStringSetting(name) {
			delete(cfg.RuntimeParams, name)
		}
	}
	cfg.RuntimeParams["standard_conforming_strings"] = "on"

	return cfg, cfg.Host, cfg.Port, domain.DbName(cfg.Database), nil
}

// apply runs every option against c in order.
func (c *connectionOptions) apply(opts ...option) {
	for _, opt := range opts {
		opt(c)
	}
}

// withReplication enables the replication=database runtime parameter, allowing
// logical replication commands. Without it the parameter is off.
func withReplication() option {
	return func(c *connectionOptions) {
		c.replication = true
	}
}

// withMaxAttempts caps connect and exec retries. Values below 1 are ignored.
func withMaxAttempts(attempts int) option {
	return func(c *connectionOptions) {
		if attempts < 1 {
			return
		}

		c.maxAttempts = attempts
	}
}

// withBaseDelay sets the wait before the first retry, which every further
// attempt doubles. Non-positive values are ignored.
func withBaseDelay(delay time.Duration) option {
	return func(c *connectionOptions) {
		if delay <= 0 {
			return
		}

		c.baseDelay = delay
	}
}

// withMaxDelay caps the per-attempt wait. Non-positive values are ignored.
func withMaxDelay(delay time.Duration) option {
	return func(c *connectionOptions) {
		if delay <= 0 {
			return
		}

		c.maxDelay = delay
	}
}

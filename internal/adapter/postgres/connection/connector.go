package connection

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"

	"hump/internal/domain"
	"hump/pkg/telemetry/log"
)

// connectionMaxAttempts caps the number of connect and exec retries before
// the operation gives up and returns the last error.
const connectionMaxAttempts = 5

var logger = log.Scope("database")

// Connector holds the connection string used to open new sessions.
// It does not own any live connection itself.
type Connector struct {
	connectionString string
}

// New returns a Connector that opens sessions to the given PostgreSQL
// connection string.
func New(connectionString string) *Connector {
	return &Connector{connectionString: connectionString}
}

// Endpoint describes the target of a connection string without its credentials.
type Endpoint struct {
	Host     string
	Port     uint16
	Database domain.DbName
}

// Describe parses connectionString and returns the target host, port, and
// database name without opening a connection. Credentials are intentionally
// omitted from the result.
func Describe(connectionString string) (Endpoint, error) {
	cfg, err := pgconn.ParseConfig(connectionString)
	if err != nil {
		return Endpoint{}, err
	}

	return Endpoint{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Database: domain.DbName(cfg.Database),
	}, nil
}

// CreateConnection opens a regular session with retry and returns a
// Connection whose Exec/ExecParams transparently reconnect on
// connection-level errors. The caller owns the connection and must Close it.
func (c *Connector) CreateConnection(ctx context.Context) (*Connection, error) {
	connection, _, err := connect(ctx, c.connectionString)
	if err != nil {
		return nil, err
	}

	return &Connection{
		connectionString: c.connectionString,
		connection:       connection,
	}, nil
}

// CreateReplicationConnection opens a session with the `replication=database`
// runtime parameter and returns the raw *pgconn.PgConn. The caller owns the
// connection and must Close it.
func (c *Connector) CreateReplicationConnection(ctx context.Context) (*pgconn.PgConn, domain.DbName, error) {
	connectionString, err := withReplication(c.connectionString)
	if err != nil {
		return nil, "", err
	}

	return connect(ctx, connectionString)
}

// connect opens a single pgconn session, retrying up to connectionMaxAttempts
// times with exponential backoff. The returned DbName is the database name
// parsed from connectionString.
func connect(ctx context.Context, connectionString string) (*pgconn.PgConn, domain.DbName, error) {
	cfg, err := pgconn.ParseConfig(connectionString)
	if err != nil {
		return nil, "", err
	}

	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		logger.Info("NOTICE: "+n.Message, slog.String("severity", n.Severity))
	}

	var lastErr error

	for attempt := range connectionMaxAttempts {
		connection, err := pgconn.ConnectConfig(ctx, cfg)
		if err == nil {
			return connection, domain.DbName(cfg.Database), nil
		}

		lastErr = err
		if attempt == connectionMaxAttempts-1 {
			break
		}

		if waitErr := backoff(ctx, attempt); waitErr != nil {
			return nil, "", waitErr
		}
	}

	return nil, "", fmt.Errorf("connect failed after %d attempts: %w", connectionMaxAttempts, lastErr)
}

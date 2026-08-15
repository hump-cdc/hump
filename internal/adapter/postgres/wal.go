package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/internal/domain"
	"hump/internal/port/database"
)

// wal owns a single logical replication session: the dedicated pgconn used to
// receive WAL data, the relation cache populated from RelationMessageV2,
// transaction-scoped emit_message context, and the user-provided process
// handler invoked for every decoded change.
type wal struct {
	// connection is the replication session handed to newWal. wal owns it and
	// closes it in Stop.
	connection *connection
	// session is the replication socket acquired once in newWal. It is never
	// re-acquired: after StartReplication the socket is in COPY-BOTH mode, so a
	// transparently reopened one would no longer be part of the stream.
	session *pgconn.PgConn

	dbName          domain.DbName
	publicationName string
	slotName        string

	pluginArguments []string

	relations map[uint32]*pglogrepl.RelationMessageV2
	typeMap   *pgtype.Map

	messageContext *postgresMessageContext

	introspector *introspector

	health  Health
	metrics *metrics
}

// standbyTimeout is the maximum wait between standby status updates sent to
// the upstream server. Receive loops break out at least this often to flush
// the current WAL position back to PostgreSQL.
const standbyTimeout = time.Second * 10

// newWal opens the replication session on conn for the given publication and
// slot and returns a wal ready to be started. conn must have been created with
// withReplication(true); wal takes ownership of it and closes it in Stop.
func newWal(
	ctx context.Context,
	conn *connection,
	publicationName string,
	slotName string,
	introspector *introspector,
	health Health,
	metricsFactory promauto.Factory,
) (*wal, error) {
	session, err := conn.Raw(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres: %w", errors.Join(err, database.ErrWal))
	}

	dbName := conn.GetDbName()

	return &wal{
		connection:      conn,
		session:         session,
		dbName:          dbName,
		publicationName: publicationName,
		slotName:        slotName,
		pluginArguments: []string{"proto_version '2'", fmt.Sprintf("publication_names '%s'", publicationName), "messages 'true'"},
		messageContext:  &postgresMessageContext{messages: map[string]string{}},
		introspector:    introspector,
		health:          health,
		metrics:         newMetrics(string(dbName), metricsFactory),
	}, nil
}

// Start begins streaming from the configured slot at LSN 0 and blocks until
// ctx is cancelled or a fatal protocol error occurs. It maintains the
// relation cache, dispatches decoded rows to the process handler, and sends
// standby status updates on every standbyTimeout tick. The last known WAL
// position is flushed before returning on any error path.
func (w *wal) Start(ctx context.Context, handler database.ProcessMessageHandler) error {
	walPosition := pglogrepl.LSN(0)

	err := pglogrepl.StartReplication(ctx, w.session, w.slotName, walPosition, pglogrepl.StartReplicationOptions{PluginArgs: w.pluginArguments})
	if err != nil {
		return fmt.Errorf("StartReplication failed: %w", errors.Join(err, database.ErrWal))
	}

	w.health.MarkHealthy()
	logger.Info("Logical replication started on slot", "slot", w.slotName)

	standbyDeadline := time.Now().Add(standbyTimeout)

	w.relations = map[uint32]*pglogrepl.RelationMessageV2{}
	w.typeMap = pgtype.NewMap()

	for {
		if time.Now().After(standbyDeadline) || ctx.Err() != nil {
			if err := w.flush(ctx, walPosition); err != nil {
				return err
			}

			standbyDeadline = time.Now().Add(standbyTimeout)
		}

		if err = ctx.Err(); err != nil {
			return err
		}

		standbyCtx, cancel := context.WithDeadline(ctx, standbyDeadline)
		rawMsg, err := w.session.ReceiveMessage(standbyCtx)
		cancel()

		if err != nil {
			if pgconn.Timeout(err) {
				continue
			}

			err = fmt.Errorf("receive message failed: %w", errors.Join(err, database.ErrWal))
			w.flush(ctx, walPosition)
			w.health.MarkUnhealthy(err)
			return err
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			err := fmt.Errorf("received Postgres WAL error: %+v: %w", errMsg, database.ErrWal)
			w.flush(ctx, walPosition)
			w.health.MarkUnhealthy(err)
			return err
		}

		msg, ok := rawMsg.(*pgproto3.CopyData)
		if !ok {
			logger.Warn("Received unexpected message", "message", fmt.Sprintf("%T", rawMsg))
			continue
		}

		switch msg.Data[0] {
		case pglogrepl.PrimaryKeepaliveMessageByteID:
			pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
			if err != nil {
				err = fmt.Errorf("ParsePrimaryKeepaliveMessage failed: %w", errors.Join(err, database.ErrWal))
				w.flush(ctx, walPosition)
				w.health.MarkUnhealthy(err)
				return err
			}

			// Keepalives carry the only fresh ServerWALEnd an idle stream gets, but
			// until the first record is decoded there is no processed position to
			// measure against and the raw server LSN would be reported as the lag.
			if walPosition != 0 {
				w.metrics.updateReplicationLag(walPosition, pkm.ServerWALEnd)
			}

			if pkm.ReplyRequested {
				standbyDeadline = time.Time{}
			}

		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
			if err != nil {
				err = fmt.Errorf("ParseXLogData failed: %w", errors.Join(err, database.ErrWal))
				w.flush(ctx, walPosition)
				w.health.MarkUnhealthy(err)
				return err
			}

			err = w.processMessage(ctx, xld.WALData, xld.ServerTime, handler)
			if err == nil {
				recordEnd := xld.WALStart + pglogrepl.LSN(len(xld.WALData))
				if recordEnd > walPosition {
					walPosition = recordEnd
				}

				w.metrics.updateReplicationLag(walPosition, xld.ServerWALEnd)
			} else {
				err = fmt.Errorf("processMessage failed: %w", errors.Join(err, database.ErrWal))
				w.flush(ctx, walPosition)
				w.health.MarkUnhealthy(err)
				return err
			}
		}
	}

}

// Stop marks the WAL sender as unhealthy and closes the underlying
// replication connection. It is safe to call once Start has returned.
func (w *wal) Stop(ctx context.Context) error {
	w.health.MarkUnhealthy(database.ErrWalClosed)
	return w.connection.Close(ctx)
}

// flush sends a standby status update reporting position as both the write
// and flush LSN, letting the upstream server advance its slot. A zero position
// is sent too: PostgreSQL ignores it for the slot but counts it as the reply
// wal_sender_timeout waits for.
func (w *wal) flush(ctx context.Context, position pglogrepl.LSN) error {
	err := pglogrepl.SendStandbyStatusUpdate(ctx, w.session,
		pglogrepl.StandbyStatusUpdate{
			WALWritePosition: position,
			WALFlushPosition: position,
		})

	if err != nil {
		err = fmt.Errorf("send status failed: %w", errors.Join(err, database.ErrWal))
		w.health.MarkUnhealthy(err)
	}

	return err
}

// processMessage parses a single pgoutput v2 message and dispatches it:
// RelationMessageV2 updates the relation cache, BeginMessage/CommitMessage
// track transaction boundaries and commit-lag metrics, LogicalDecodingMessageV2
// is buffered as transactional context, and row-change messages are decoded
// and forwarded to the process handler.
func (w *wal) processMessage(ctx context.Context, walData []byte, timestamp time.Time, handler database.ProcessMessageHandler) error {
	message, err := pglogrepl.ParseV2(walData, false)
	if err != nil {
		return fmt.Errorf("Parse logical replication message: %w", err)
	}

	switch message := message.(type) {
	case *pglogrepl.RelationMessageV2:
		w.relations[message.RelationID] = message

	case *pglogrepl.CommitMessage:
		w.messageContext.flush()

	case *pglogrepl.LogicalDecodingMessageV2:
		if !message.Transactional || len(message.Prefix) == 0 || len(message.Content) == 0 {
			return nil
		}

		w.messageContext.append(message.Prefix, string(message.Content))

	case *pglogrepl.InsertMessageV2:
		start := time.Now()
		relation, ok := w.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("unknown relation ID: %d: %w", message.RelationID, database.ErrWal)
		}

		prev, next, err := w.decodeRow(nil, message.Tuple, relation)
		if err != nil {
			return fmt.Errorf("decode insert tuples for %s.%s: %w", relation.Namespace, relation.RelationName, err)
		}

		headers := w.messageContext.dump()
		table := domain.TableIdentifier{Schema: domain.TableSchema(relation.Namespace), Name: domain.TableName(relation.RelationName)}
		ref, _, err := w.introspector.ResolveTableConfiguration(ctx, table)
		if errors.Is(err, ErrTableIgnored) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", table.Schema, table.Name, err)
		}

		err = handler(ctx, timestamp, domain.OperationTypeInsert, w.dbName, ref.Schema, ref.Name, headers, prev, next)
		if err != nil {
			return err
		}

		elapsed := time.Since(start)
		w.metrics.observeInsert(ref, elapsed)

	case *pglogrepl.UpdateMessageV2:
		start := time.Now()
		relation, ok := w.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("unknown relation ID: %d: %w", message.RelationID, database.ErrWal)
		}

		prev, next, err := w.decodeRow(message.OldTuple, message.NewTuple, relation)
		if err != nil {
			return fmt.Errorf("decode update tuples for %s.%s: %w", relation.Namespace, relation.RelationName, err)
		}

		headers := w.messageContext.dump()
		table := domain.TableIdentifier{Schema: domain.TableSchema(relation.Namespace), Name: domain.TableName(relation.RelationName)}
		ref, _, err := w.introspector.ResolveTableConfiguration(ctx, table)
		if errors.Is(err, ErrTableIgnored) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", table.Schema, table.Name, err)
		}

		err = handler(ctx, timestamp, domain.OperationTypeUpdate, w.dbName, ref.Schema, ref.Name, headers, prev, next)
		if err != nil {
			return err
		}

		elapsed := time.Since(start)
		w.metrics.observeUpdate(ref, elapsed)

	case *pglogrepl.DeleteMessageV2:
		start := time.Now()
		relation, ok := w.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("unknown relation ID: %d: %w", message.RelationID, database.ErrWal)
		}

		prev, next, err := w.decodeRow(message.OldTuple, nil, relation)
		if err != nil {
			return fmt.Errorf("decode delete tuples for %s.%s: %w", relation.Namespace, relation.RelationName, err)
		}

		headers := w.messageContext.dump()
		table := domain.TableIdentifier{Schema: domain.TableSchema(relation.Namespace), Name: domain.TableName(relation.RelationName)}
		ref, _, err := w.introspector.ResolveTableConfiguration(ctx, table)
		if errors.Is(err, ErrTableIgnored) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", table.Schema, table.Name, err)
		}

		err = handler(ctx, timestamp, domain.OperationTypeDelete, w.dbName, ref.Schema, ref.Name, headers, prev, next)
		if err != nil {
			return err
		}

		elapsed := time.Since(start)
		w.metrics.observeDelete(ref, elapsed)

	case *pglogrepl.TruncateMessageV2:
		resultErrs := make([]error, 0, len(message.RelationIDs))
		headers := w.messageContext.dump()
		for _, relationId := range message.RelationIDs {
			start := time.Now()
			relation, ok := w.relations[relationId]
			if !ok {
				return fmt.Errorf("unknown relation ID: %d: %w", relationId, database.ErrWal)
			}

			table := domain.TableIdentifier{Schema: domain.TableSchema(relation.Namespace), Name: domain.TableName(relation.RelationName)}
			ref, _, err := w.introspector.ResolveTableConfiguration(ctx, table)
			if errors.Is(err, ErrTableIgnored) {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", table.Schema, table.Name, err)
			}

			e := handler(ctx, timestamp, domain.OperationTypeTruncate, w.dbName, ref.Schema, ref.Name, headers, nil, nil)
			if e != nil {
				resultErrs = append(resultErrs, e)
			} else {
				elapsed := time.Since(start)
				w.metrics.observeTruncate(ref, elapsed)
			}
		}

		if len(resultErrs) > 0 {
			return errors.Join(resultErrs...)
		}
	}

	return nil
}

// decodeRow turns the optional previous and next tuples of a change event
// into column maps using relation for column names and OIDs. Unchanged TOAST
// columns ('u') are dropped from prev and copied from prev into next.
func (w *wal) decodeRow(prevRow, nextRow *pglogrepl.TupleData, relation *pglogrepl.RelationMessageV2) (domain.DbRecord, domain.DbRecord, error) {
	var prev domain.DbRecord
	if prevRow != nil {
		prev = domain.DbRecord{}

		for index, column := range prevRow.Columns {
			columnName := relation.Columns[index].Name

			switch column.DataType {
			case 'n': // null
				prev[columnName] = nil
			case 'u': // unchanged toast: ignore on prev
			case 't': // text
				val, err := w.decodeRowText(column.Data, relation.Columns[index].DataType)
				if err != nil {
					return nil, nil, fmt.Errorf("decode column %q (oid %d): %w", columnName, relation.Columns[index].DataType, err)
				}
				prev[columnName] = val
			}
		}
	}

	var next domain.DbRecord
	if nextRow != nil {
		next = domain.DbRecord{}

		for index, column := range nextRow.Columns {
			columnName := relation.Columns[index].Name

			switch column.DataType {
			case 'n': // null
				next[columnName] = nil
			case 'u': // unchanged toast
				if val, ok := prev[columnName]; ok {
					next[columnName] = val
				}
			case 't': // text
				val, err := w.decodeRowText(column.Data, relation.Columns[index].DataType)
				if err != nil {
					return nil, nil, fmt.Errorf("decode column %q (oid %d): %w", columnName, relation.Columns[index].DataType, err)
				}
				next[columnName] = val
			}
		}
	}

	return prev, next, nil
}

// decodeRowText decodes a text-format column value using the pgtype codec
// registered for dataType, falling back to the raw string when the OID is
// unknown to the type map.
func (w *wal) decodeRowText(data []byte, dataType uint32) (any, error) {
	dt, ok := w.typeMap.TypeForOID(dataType)
	if !ok {
		return string(data), nil
	}

	val, err := dt.Codec.DecodeValue(w.typeMap, dataType, pgtype.TextFormatCode, data)
	if err != nil {
		return nil, err
	}

	if b, ok := val.([16]byte); ok && dataType == pgtype.UUIDOID {
		return uuidString(b), nil
	}

	return val, nil
}

// uuidString formats a raw 16-byte UUID as its canonical
// 8-4-4-4-12 lowercase hex string.
func uuidString(b [16]byte) string {
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])

	return string(buf[:])
}

// postgresMessageContext is a thread-safe store for messages emitted via
// pg_logical_emit_message() and decoded from the WAL stream. It buffers
// name/value pairs that arrive between transaction boundaries so they can be
// attached to subsequent change events.
type postgresMessageContext struct {
	mu       sync.Mutex
	messages map[string]string
}

// append stores a message under the given name, overwriting any prior value.
func (c *postgresMessageContext) append(name, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages[name] = value
}

// flush returns the buffered messages and resets the internal store.
func (c *postgresMessageContext) flush() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := c.messages
	c.messages = map[string]string{}

	return messages
}

// dump returns a copy of the buffered messages without clearing the store.
func (c *postgresMessageContext) dump() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make(map[string]string, len(c.messages))
	maps.Copy(messages, c.messages)

	return messages
}

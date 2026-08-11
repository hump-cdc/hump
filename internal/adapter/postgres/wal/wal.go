// Package wal consumes the PostgreSQL logical replication stream (pgoutput,
// protocol v2). It maintains the relation cache, decodes INSERT/UPDATE/
// DELETE/TRUNCATE tuples into Go values, attaches buffered
// pg_logical_emit_message context to each change, and periodically sends
// standby status updates so the upstream slot can advance.
package wal

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"

	"hump/internal/adapter/postgres/connection"
	"hump/internal/adapter/postgres/introspection"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/types/errs"
	"hump/pkg/telemetry/log"
)

// Wal owns a single logical replication session: the dedicated pgconn used to
// receive WAL data, the relation cache populated from RelationMessageV2,
// transaction-scoped emit_message context, and the user-provided process
// handler invoked for every decoded change.
type Wal struct {
	connection *pgconn.PgConn

	dbName          domain.DbName
	publicationName string
	slotName        string

	pluginArguments []string

	relations map[uint32]*pglogrepl.RelationMessageV2
	typeMap   *pgtype.Map

	messageContext *messageContext

	lastCommitTime time.Time

	introspector *introspection.Introspector
	// health         *health.Health
	// metrics        *metrics.Metrics

	// process wal.ProcessHandler
}

var logger = log.Scope("WAL")

// standbyTimeout is the maximum wait between standby status updates sent to
// the upstream server. Receive loops break out at least this often to flush
// the current WAL position back to PostgreSQL.
const standbyTimeout = time.Second * 10

// New opens a replication-mode session for the given publication and slot
// and returns a Wal ready to be started. The handler is invoked for every
// decoded INSERT, UPDATE, DELETE or TRUNCATE change.
func New(
	ctx context.Context,
	connector *connection.Connector,
	publicationName string,
	slotName string,
	introspector *introspection.Introspector,
	// health *health.Health,
	// metrics *metrics.Metrics,
	// process wal.ProcessHandler
) (*Wal, error) {
	connection, dbName, err := connector.CreateReplicationConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres: %w", errors.Join(err, errs.ErrWal))
	}

	return &Wal{
		connection:      connection,
		dbName:          dbName,
		publicationName: publicationName,
		slotName:        slotName,
		pluginArguments: []string{
			"proto_version '2'",
			fmt.Sprintf("publication_names '%s'", publicationName),
			"messages 'true'",
		},
		messageContext: &messageContext{messages: map[string]string{}},
		lastCommitTime: time.Time{},
		introspector:   introspector,
		// health:         health,
		// metrics:        metrics,
		// process: process,
	}, nil
}

// Start begins streaming from the configured slot at LSN 0 and blocks until
// ctx is cancelled or a fatal protocol error occurs. It maintains the
// relation cache, dispatches decoded rows to the process handler, and sends
// standby status updates on every standbyTimeout tick. The last known WAL
// position is flushed before returning on any error path.
func (w *Wal) Start(ctx context.Context, handler database.ProcessMessageHandler) error {
	walPosition := pglogrepl.LSN(0)

	err := pglogrepl.StartReplication(ctx, w.connection, w.slotName, walPosition, pglogrepl.StartReplicationOptions{PluginArgs: w.pluginArguments})
	if err != nil {
		return fmt.Errorf("StartReplication failed: %w", errors.Join(err, errs.ErrWal))
	}

	// TODO: w.health.WalSender.Update(true, nil)
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
		rawMsg, err := w.connection.ReceiveMessage(standbyCtx)
		cancel()

		if err != nil {
			if pgconn.Timeout(err) {
				continue
			}

			err = fmt.Errorf("receive message failed: %w", errors.Join(err, errs.ErrWal))
			// TODO: w.health.WalSender.Update(false, err)
			w.flush(ctx, walPosition)
			return err
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			err := fmt.Errorf("received Postgres WAL error: %+v: %w", errMsg, errs.ErrWal)
			// TODO: w.health.WalSender.Update(false, err)
			w.flush(ctx, walPosition)
			return err
		}

		if w.lastCommitTime.IsZero() {
			// TODO: w.metrics.LagCommit(0)
		} else {
			// TODO: w.metrics.LagCommit(float64(time.Since(w.lastCommitTime).Seconds()))
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
				err = fmt.Errorf("ParsePrimaryKeepaliveMessage failed: %w", errors.Join(err, errs.ErrWal))
				// TODO: w.health.WalSender.Update(false, err)
				w.flush(ctx, walPosition)
				return err
			}

			// TODO: w.metrics.LagSender(time.Since(pkm.ServerTime).Seconds())

			if pkm.ReplyRequested {
				standbyDeadline = time.Time{}
			}

		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
			if err != nil {
				err = fmt.Errorf("ParseXLogData failed: %w", errors.Join(err, errs.ErrWal))
				// TODO: w.health.WalSender.Update(false, err)
				w.flush(ctx, walPosition)
				return err
			}

			err = w.processMessage(ctx, xld.WALData, xld.ServerTime, handler)
			if err == nil {
				recordEnd := xld.WALStart + pglogrepl.LSN(len(xld.WALData))
				if recordEnd > walPosition {
					walPosition = recordEnd
				}

				// TODO: w.metrics.LagSender(time.Since(xld.ServerTime).Seconds())
			} else {
				err = fmt.Errorf("processMessage failed: %w", errors.Join(err, errs.ErrWal))
				// TODO: w.health.WalSender.Update(false, err)
				w.flush(ctx, walPosition)
				return err
			}
		}
	}

}

// Stop marks the WAL sender as unhealthy and closes the underlying
// replication connection. It is safe to call once Start has returned.
func (w *Wal) Stop(ctx context.Context) error {
	// TODO: w.health.WalSender.Update(false, errors.New("wal sender connection closed"))
	return w.connection.Close(ctx)
}

// flush sends a standby status update reporting position as both the write
// and flush LSN, letting the upstream server advance its slot.
func (w *Wal) flush(ctx context.Context, position pglogrepl.LSN) error {
	err := pglogrepl.SendStandbyStatusUpdate(ctx, w.connection, pglogrepl.StandbyStatusUpdate{
		WALWritePosition: position,
		WALFlushPosition: position,
	})

	if err != nil {
		err = fmt.Errorf("send status failed: %w", errors.Join(err, errs.ErrWal))
		// TODO: w.health.WalSender.Update(false, err)
	}

	return err
}

// processMessage parses a single pgoutput v2 message and dispatches it:
// RelationMessageV2 updates the relation cache, BeginMessage/CommitMessage
// track transaction boundaries and commit-lag metrics, LogicalDecodingMessageV2
// is buffered as transactional context, and row-change messages are decoded
// and forwarded to the process handler.
func (w *Wal) processMessage(ctx context.Context, walData []byte, timestamp time.Time, handler database.ProcessMessageHandler) error {
	message, err := pglogrepl.ParseV2(walData, false)
	if err != nil {
		return fmt.Errorf("Parse logical replication message: %w", err)
	}

	switch message := message.(type) {
	case *pglogrepl.RelationMessageV2:
		w.relations[message.RelationID] = message

	case *pglogrepl.BeginMessage:
		w.lastCommitTime = message.CommitTime
		// TODO: w.metrics.LagCommit(time.Since(w.lastCommitTime).Seconds())

	case *pglogrepl.CommitMessage:
		w.lastCommitTime = time.Time{}
		// TODO: w.metrics.LagCommit(0)

		w.messageContext.flush()

	case *pglogrepl.LogicalDecodingMessageV2:
		if !message.Transactional || len(message.Prefix) == 0 || len(message.Content) == 0 {
			return nil
		}

		w.messageContext.append(message.Prefix, string(message.Content))

	case *pglogrepl.InsertMessageV2:

		// TODO: start := time.Now()
		relation, ok := w.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("unknown relation ID: %d: %w", message.RelationID, errs.ErrWal)
		}

		prev, next, err := w.decodeRow(nil, message.Tuple, relation)
		if err != nil {
			return fmt.Errorf("decode insert tuples for %s.%s: %w", relation.Namespace, relation.RelationName, err)
		}

		headers := w.messageContext.dump()
		ref, _, err := w.introspector.ResolveTableConfiguration(ctx, domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName))
		if err != nil {
			return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName), err)
		}

		err = handler(ctx, timestamp, domain.OperationTypeInsert, w.dbName, ref.Schema, ref.Name, headers, prev, next)
		if err != nil {
			return err
		}

		// TODO: w.metrics.ProcessedRowsInsertDuration(time.Since(start).Seconds(), dbName, schema, table)

	case *pglogrepl.UpdateMessageV2:
		// TODO: start := time.Now()
		relation, ok := w.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("unknown relation ID: %d: %w", message.RelationID, errs.ErrWal)
		}

		prev, next, err := w.decodeRow(message.OldTuple, message.NewTuple, relation)
		if err != nil {
			return fmt.Errorf("decode update tuples for %s.%s: %w", relation.Namespace, relation.RelationName, err)
		}

		headers := w.messageContext.dump()
		ref, _, err := w.introspector.ResolveTableConfiguration(ctx, domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName))
		if err != nil {
			return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName), err)
		}

		err = handler(ctx, timestamp, domain.OperationTypeUpdate, w.dbName, ref.Schema, ref.Name, headers, prev, next)
		if err != nil {
			return err
		}

		// TODO: w.metrics.ProcessedRowsUpdateDuration(time.Since(start).Seconds(), dbName, schema, table)

	case *pglogrepl.DeleteMessageV2:
		// TODO: start := time.Now()
		relation, ok := w.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("unknown relation ID: %d: %w", message.RelationID, errs.ErrWal)
		}

		prev, next, err := w.decodeRow(message.OldTuple, nil, relation)
		if err != nil {
			return fmt.Errorf("decode delete tuples for %s.%s: %w", relation.Namespace, relation.RelationName, err)
		}

		headers := w.messageContext.dump()
		ref, _, err := w.introspector.ResolveTableConfiguration(ctx, domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName))
		if err != nil {
			return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName), err)
		}

		err = handler(ctx, timestamp, domain.OperationTypeDelete, w.dbName, ref.Schema, ref.Name, headers, prev, next)
		if err != nil {
			return err
		}

		// TODO: w.metrics.ProcessedRowsDeleteDuration(time.Since(start).Seconds(), dbName, schema, table)

	case *pglogrepl.TruncateMessageV2:
		resultErrs := make([]error, 0, len(message.RelationIDs))
		headers := w.messageContext.dump()
		for _, relationId := range message.RelationIDs {
			// TODO: start := time.Now()
			relation, ok := w.relations[relationId]
			if !ok {
				return fmt.Errorf("unknown relation ID: %d: %w", relationId, errs.ErrWal)
			}

			ref, _, err := w.introspector.ResolveTableConfiguration(ctx, domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName))
			if err != nil {
				return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", domain.TableSchema(relation.Namespace), domain.TableName(relation.RelationName), err)
			}

			e := handler(ctx, timestamp, domain.OperationTypeTruncate, w.dbName, ref.Schema, ref.Name, headers, nil, nil)
			if e != nil {
				resultErrs = append(resultErrs, e)
			} else {
				// TODO: w.metrics.ProcessedRowsTruncateDuration(time.Since(start).Seconds(), dbName, schema, table)
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
func (w *Wal) decodeRow(prevRow, nextRow *pglogrepl.TupleData, relation *pglogrepl.RelationMessageV2) (domain.DbRecord, domain.DbRecord, error) {
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
func (w *Wal) decodeRowText(data []byte, dataType uint32) (any, error) {
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

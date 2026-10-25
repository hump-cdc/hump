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

	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/pkg/app/shutdown"
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

	// inTransaction is set between Begin and Commit, where a keepalive's
	// ServerWALEnd may already lie past rows not yet processed.
	inTransaction bool

	introspector *introspector

	// gate pauses delivery to the handler while a snapshot runs; nil never pauses.
	gate *walGate

	// standbyInterval is the wait between standby status updates. It is
	// standbyTimeout outside tests, which shorten it; a zero value would panic
	// the keepAlive ticker and spin the receive loop.
	standbyInterval time.Duration

	// processing is set while the receive loop is inside processMessage, the only
	// time the keepalive worker writes to session; processingPosition is what it
	// reports then. processingMu guards both and is held across a worker flush.
	processingMu       sync.Mutex
	processing         bool
	processingPosition pglogrepl.LSN

	// shutdownAt is when Start noticed a shutdown; zero when it ended otherwise.
	shutdownAt time.Time

	health  Health
	metrics *metrics
}

// standbyTimeout is the maximum wait between standby status updates sent to
// the upstream server. Receive loops break out at least this often to flush
// the current WAL position back to PostgreSQL.
const standbyTimeout = time.Second * 10

// newWal opens the replication session on conn for the given publication and
// slot and returns a wal ready to be started. conn must have been created with
// withReplication(); wal takes ownership of it and closes it in Stop. m is
// owned by the caller and shared across sessions; a nil gate never pauses.
func newWal(
	ctx context.Context,
	conn *connection,
	publicationName string,
	slotName string,
	introspector *introspector,
	gate *walGate,
	health Health,
	m *metrics,
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
		gate:            gate,
		standbyInterval: standbyTimeout,
		health:          health,
		metrics:         m,
	}, nil
}

// Start begins streaming from the configured slot at LSN 0 and blocks until
// ctx is cancelled or a fatal protocol error occurs. Cancelling ctx between
// transactions stops at once; inside one, the rest of it is received and
// processed on a detached context so its commit can be confirmed. The WAL
// position is flushed on every standby tick, while a change is processed, and
// on exit.
func (w *wal) Start(ctx context.Context, handler database.ProcessMessageHandler) error {
	walPosition := pglogrepl.LSN(0)
	workCtx := context.WithoutCancel(ctx)

	if w.gate != nil {
		handler = w.gate.wrap(ctx, handler)
	}

	err := pglogrepl.StartReplication(ctx, w.session, w.slotName, walPosition, pglogrepl.StartReplicationOptions{PluginArgs: w.pluginArguments})
	if err != nil {
		return fmt.Errorf("StartReplication failed: %w", errors.Join(err, database.ErrWal))
	}

	w.health.MarkHealthy()
	logger.Info("Logical replication started on slot", "slot", w.slotName)

	// The receive deadline is the standby tick, so the context is built once and
	// rebuilt only when that tick is reset. Per message it would cost a runtime
	// timer plus a contextWatcher registration inside ReceiveMessage.
	standbyCtx, cancelStandby := context.WithTimeout(ctx, w.standbyInterval)
	defer func() { cancelStandby() }()

	// draining is set once ctx is cancelled inside a transaction: receive windows
	// then open on workCtx until its commit arrives.
	draining := false

	// resetStandby cancels the previous receive context and opens the next window.
	// It stays a closure so vet's lostcancel sees one defended assignment.
	resetStandby := func() {
		cancelStandby()

		parent := ctx
		if draining {
			parent = workCtx
		}

		standbyCtx, cancelStandby = context.WithTimeout(parent, w.standbyInterval)
	}

	w.relations = map[uint32]*pglogrepl.RelationMessageV2{}
	w.typeMap = pgtype.NewMap()
	w.inTransaction = false

	stopKeepAlive := w.keepAlive(workCtx)
	defer stopKeepAlive()

	shutdown.OnShutdown(shutdown.StageWorkers, "wal_position", w.onShutdown)

	for {
		rawMsg, err := w.session.ReceiveMessage(standbyCtx)

		if err != nil {
			// A cancelled ctx means "take no new work", and the rest of an open
			// transaction is not new: its commit, confirmed on exit, is what keeps
			// the rows already delivered from being replayed after a restart.
			if !draining && w.inTransaction && ctx.Err() != nil {
				draining = true
				resetStandby()

				continue
			}

			// The receive deadline is the standby tick, so a timeout is the send
			// window closing, not a broken stream. A finished ctx surfaces either way —
			// pgx reports a cancelled one as a bare context.Canceled and an expired
			// deadline as a timeout — and fail tells the two apart from the real faults.
			if pgconn.Timeout(err) {
				if ctxErr := ctx.Err(); ctxErr != nil && !draining {
					return w.fail(ctx, walPosition, ctxErr)
				}

				if err := w.flush(ctx, walPosition); err != nil {
					return err
				}

				resetStandby()

				continue
			}

			err = fmt.Errorf("receive message failed: %w", errors.Join(err, database.ErrWal))
			return w.fail(ctx, walPosition, err)
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			err := fmt.Errorf("received Postgres WAL error: %+v: %w", errMsg, database.ErrWal)
			return w.fail(ctx, walPosition, err)
		}

		msg, ok := rawMsg.(*pgproto3.CopyData)
		if !ok {
			logger.Warn("Received unexpected message", "message", fmt.Sprintf("%T", rawMsg))
			continue
		}

		// A conforming sender never frames an empty body, so the stream is no
		// longer one this loop can decode.
		if len(msg.Data) == 0 {
			err := errors.Join(errEmptyCopyData, database.ErrWal)
			return w.fail(ctx, walPosition, err)
		}

		switch msg.Data[0] {
		case pglogrepl.PrimaryKeepaliveMessageByteID:
			pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
			if err != nil {
				err = fmt.Errorf("ParsePrimaryKeepaliveMessage failed: %w", errors.Join(err, database.ErrWal))
				return w.fail(ctx, walPosition, err)
			}

			// Between transactions everything committed before ServerWALEnd has
			// already been processed, so confirming it releases the WAL pgoutput
			// skipped. Inside one the sender may have read past this commit.
			if !w.inTransaction && pkm.ServerWALEnd > walPosition {
				walPosition = pkm.ServerWALEnd
			}

			if pkm.ReplyRequested {
				if err := w.flush(ctx, walPosition); err != nil {
					return err
				}

				resetStandby()
			}

		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
			if err != nil {
				err = fmt.Errorf("ParseXLogData failed: %w", errors.Join(err, database.ErrWal))
				return w.fail(ctx, walPosition, err)
			}

			w.beginProcessing(walPosition)
			commitEnd, err := w.processMessage(workCtx, xld.WALData, xld.ServerTime, handler)
			w.endProcessing()

			if err == nil {
				if commitEnd > walPosition {
					walPosition = commitEnd
				}
			} else {
				err = fmt.Errorf("processMessage failed: %w", errors.Join(err, database.ErrWal))
				return w.fail(ctx, walPosition, err)
			}

			if ctxErr := ctx.Err(); ctxErr != nil && !w.inTransaction {
				return w.fail(ctx, walPosition, ctxErr)
			}
		}
	}

}

// Stop marks the WAL sender as unhealthy and closes the underlying
// replication connection, logging the stop when Start ended on a shutdown.
// It is safe to call once Start has returned.
func (w *wal) Stop(ctx context.Context) error {
	w.health.MarkUnhealthy(database.ErrWalClosed)

	if err := w.connection.Close(ctx); err != nil {
		return err
	}

	if !w.shutdownAt.IsZero() {
		logger.Info("Service stopped", "name", "wal", "duration", time.Since(w.shutdownAt))
	}

	return nil
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

// onShutdown confirms the last processed position when shutdown finds a change
// still in flight, as after a drain timeout: Start never flushes on its way out
// then, since the process exits first. Otherwise Start has flushed already.
func (w *wal) onShutdown(ctx context.Context) error {
	w.processingMu.Lock()
	defer w.processingMu.Unlock()

	if !w.processing {
		return nil
	}

	return w.flush(ctx, w.processingPosition)
}

// fail flushes position back to the server on a best-effort basis, marks the
// sender unhealthy with err and returns it. The flush error is dropped: err is
// why the stream is ending. A finished ctx is the exception — err then only
// restates the shutdown, so health is left alone and ctx.Err() is returned.
func (w *wal) fail(ctx context.Context, position pglogrepl.LSN, err error) error {
	// The stream is ending because the process is: pgx normalizes a cancelled
	// receive into a bare context.Canceled. Requiring err to carry the context
	// error keeps a real fault, such as a handler failing during drain, visible.
	if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
		w.shutdownAt = time.Now()

		// Detached so the last position still goes out. It reaches the socket today
		// because pglogrepl.SendStandbyStatusUpdate ignores the context it takes.
		_ = w.flush(context.WithoutCancel(ctx), position)

		return ctxErr
	}

	_ = w.flush(ctx, position)
	w.health.MarkUnhealthy(err)

	return err
}

// resolveTable resolves table to the configured table that owns it, going to
// the database only when the registry cannot answer.
func (w *wal) resolveTable(ctx context.Context, table domain.TableIdentifier) (domain.TableIdentifier, error) {
	ref, err := w.introspector.ResolveKnownTableConfiguration(table)
	if !errors.Is(err, errIntrospectionRequired) {
		return ref, err
	}

	return w.introspector.ResolveTableConfiguration(ctx, table)
}

// keepAlive starts the worker that sends a standby status update every
// standbyInterval while a message is being processed, so a slow sink or a
// blocking introspection cannot let the upstream walsender time the session
// out. The returned stop waits for the worker to exit.
func (w *wal) keepAlive(ctx context.Context) (stop func()) {
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		ticker := time.NewTicker(w.standbyInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				w.processingMu.Lock()
				if w.processing {
					// A failure here already marks the sender unhealthy; the receive
					// loop reports it once it regains control.
					_ = w.flush(ctx, w.processingPosition)
				}
				w.processingMu.Unlock()
			}
		}
	}()

	return func() {
		close(done)
		<-finished
	}
}

// beginProcessing hands session to the keepalive worker, which reports position
// until endProcessing takes it back.
func (w *wal) beginProcessing(position pglogrepl.LSN) {
	w.processingMu.Lock()
	defer w.processingMu.Unlock()

	w.processing = true
	w.processingPosition = position
}

// endProcessing takes session back from the keepalive worker, waiting out a
// flush it has in flight, so the receive loop stays its only user.
func (w *wal) endProcessing() {
	w.processingMu.Lock()
	defer w.processingMu.Unlock()

	w.processing = false
}

// processMessage parses a single pgoutput v2 message and dispatches it: a
// relation updates the cache, a transactional logical message is buffered as
// context, a row change is decoded and forwarded to handler, and a commit drops
// the buffer and returns the transaction's end LSN; any other message returns 0.
func (w *wal) processMessage(ctx context.Context, walData []byte, timestamp time.Time, handler database.ProcessMessageHandler) (pglogrepl.LSN, error) {
	if len(walData) == 0 {
		return 0, errEmptyWalRecord
	}

	message, err := pglogrepl.ParseV2(walData, false)
	if err != nil {
		return 0, fmt.Errorf("Parse logical replication message: %w", err)
	}

	switch message := message.(type) {
	case *pglogrepl.RelationMessageV2:
		w.relations[message.RelationID] = message

	case *pglogrepl.BeginMessage:
		w.inTransaction = true

	case *pglogrepl.CommitMessage:
		w.messageContext.reset()
		w.inTransaction = false

		return message.TransactionEndLSN, nil

	case *pglogrepl.LogicalDecodingMessageV2:
		if !message.Transactional || len(message.Prefix) == 0 || len(message.Content) == 0 {
			return 0, nil
		}

		w.messageContext.append(message.Prefix, string(message.Content))

	case *pglogrepl.InsertMessageV2:
		return 0, w.processRow(ctx, rowChange{
			operation:     domain.OperationTypeInsert,
			relationID:    message.RelationID,
			prevTupleType: pglogrepl.UpdateMessageTupleTypeNone,
			nextTuple:     message.Tuple,
		}, timestamp, handler)

	case *pglogrepl.UpdateMessageV2:
		return 0, w.processRow(ctx, rowChange{
			operation:     domain.OperationTypeUpdate,
			relationID:    message.RelationID,
			prevTuple:     message.OldTuple,
			prevTupleType: message.OldTupleType,
			nextTuple:     message.NewTuple,
		}, timestamp, handler)

	case *pglogrepl.DeleteMessageV2:
		return 0, w.processRow(ctx, rowChange{
			operation:     domain.OperationTypeDelete,
			relationID:    message.RelationID,
			prevTuple:     message.OldTuple,
			prevTupleType: message.OldTupleType,
		}, timestamp, handler)

	case *pglogrepl.TruncateMessageV2:
		resultErrs := make([]error, 0, len(message.RelationIDs))
		headers := w.messageContext.dump()
		for _, relationId := range message.RelationIDs {
			start := time.Now()
			relation, ok := w.relations[relationId]
			if !ok {
				return 0, fmt.Errorf("unknown relation ID: %d: %w", relationId, database.ErrWal)
			}

			table := domain.TableIdentifier{Schema: domain.TableSchema(relation.Namespace), Name: domain.TableName(relation.RelationName)}
			ref, err := w.resolveTable(ctx, table)
			if errors.Is(err, ErrTableIgnored) {
				continue
			}
			if err != nil {
				return 0, fmt.Errorf("failed to resolve table configuration for %s.%s: %w", table.Schema, table.Name, err)
			}

			e := handler(ctx, timestamp, domain.OperationTypeTruncate, w.dbName, ref.Schema, ref.Name, headers, nil, nil)
			if e != nil {
				resultErrs = append(resultErrs, e)
			} else {
				elapsed := time.Since(start)
				w.metrics.observeRow(domain.OperationTypeTruncate, ref, elapsed)
			}
		}

		if len(resultErrs) > 0 {
			return 0, errors.Join(resultErrs...)
		}
	}

	return 0, nil
}

// rowChange is the per-operation part of a decoded row change: the tuples that
// go into decodeRow and the operation reported to the handler and to metrics.
type rowChange struct {
	// operation is the change kind reported to the handler and used as the
	// metric label.
	operation  domain.OperationType
	relationID uint32
	// prevTuple is the old tuple, nil when the operation carries none.
	prevTuple *pglogrepl.TupleData
	// prevTupleType is the old tuple's pgoutput kind.
	prevTupleType uint8
	// nextTuple is the new tuple, nil for a delete.
	nextTuple *pglogrepl.TupleData
}

// processRow resolves the relation behind change, decodes its tuples and
// forwards the result to handler, recording the time the whole path took.
func (w *wal) processRow(ctx context.Context, change rowChange, timestamp time.Time, handler database.ProcessMessageHandler) error {
	start := time.Now()

	relation, ok := w.relations[change.relationID]
	if !ok {
		return fmt.Errorf("unknown relation ID: %d: %w", change.relationID, database.ErrWal)
	}

	table := domain.TableIdentifier{Schema: domain.TableSchema(relation.Namespace), Name: domain.TableName(relation.RelationName)}
	ref, err := w.resolveTable(ctx, table)
	if errors.Is(err, ErrTableIgnored) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to resolve table configuration for %s.%s: %w", table.Schema, table.Name, err)
	}

	prev, next, err := w.decodeRow(change.prevTuple, change.prevTupleType, change.nextTuple, relation)
	if err != nil {
		return fmt.Errorf("decode %s tuples for %s.%s: %w", change.operation, relation.Namespace, relation.RelationName, err)
	}

	headers := w.messageContext.dump()

	err = handler(ctx, timestamp, change.operation, w.dbName, ref.Schema, ref.Name, headers, prev, next)
	if err != nil {
		return err
	}

	w.metrics.observeRow(change.operation, ref, time.Since(start))

	return nil
}

// decodeRow turns the optional previous and next tuples of a change event into
// column maps using relation for column names and OIDs. prevTupleType is the
// old tuple's pgoutput kind. Columns the stream did not carry are reported as
// [domain.Unavailable] rather than dropped or reported as NULL.
func (w *wal) decodeRow(prevRow *pglogrepl.TupleData, prevTupleType uint8, nextRow *pglogrepl.TupleData, relation *pglogrepl.RelationMessageV2) (domain.DbRecord, domain.DbRecord, error) {
	var prev domain.DbRecord
	if prevRow != nil {
		if len(prevRow.Columns) != len(relation.Columns) {
			return nil, nil, fmt.Errorf("old tuple has %d columns, relation has %d", len(prevRow.Columns), len(relation.Columns))
		}

		prev = domain.DbRecord{}

		for index, column := range prevRow.Columns {
			columnName := relation.Columns[index].Name

			switch column.DataType {
			case 'n': // null
				// A key-only tuple nulls out everything outside the replica
				// identity, so those nulls are absences, not values. Flags == 1
				// means "part of the replica identity" — under FULL every column
				// carries it, which is why the key-only check gates on the tuple
				// type rather than on the flag alone.
				if prevTupleType == pglogrepl.UpdateMessageTupleTypeKey && relation.Columns[index].Flags != 1 {
					prev[columnName] = domain.Unavailable
					continue
				}
				prev[columnName] = nil
			case 'u': // unchanged toast: the old value is not in the stream
				prev[columnName] = domain.Unavailable
			case 't': // text
				val, err := decodeText(w.typeMap, relation.Columns[index].DataType, column.Data)
				if err != nil {
					return nil, nil, fmt.Errorf("decode column %q (oid %d): %w", columnName, relation.Columns[index].DataType, err)
				}
				prev[columnName] = val
			default:
				return nil, nil, fmt.Errorf("unexpected tuple data type %q for column %q", column.DataType, columnName)
			}
		}
	}

	var next domain.DbRecord
	if nextRow != nil {
		if len(nextRow.Columns) != len(relation.Columns) {
			return nil, nil, fmt.Errorf("new tuple has %d columns, relation has %d", len(nextRow.Columns), len(relation.Columns))
		}

		next = domain.DbRecord{}

		for index, column := range nextRow.Columns {
			columnName := relation.Columns[index].Name

			switch column.DataType {
			case 'n': // null
				next[columnName] = nil
			case 'u': // unchanged toast
				if val, ok := prev[columnName]; ok {
					next[columnName] = val
				} else {
					next[columnName] = domain.Unavailable
				}
			case 't': // text
				val, err := decodeText(w.typeMap, relation.Columns[index].DataType, column.Data)
				if err != nil {
					return nil, nil, fmt.Errorf("decode column %q (oid %d): %w", columnName, relation.Columns[index].DataType, err)
				}
				next[columnName] = val
			default:
				return nil, nil, fmt.Errorf("unexpected tuple data type %q for column %q", column.DataType, columnName)
			}
		}
	}

	return prev, next, nil
}

// decodeText decodes a text-format column value using the pgtype codec
// registered in m for oid, falling back to the raw string when the OID is
// unknown to m. A date or timestamp infinity comes back as an infiniteTime.
func decodeText(m *pgtype.Map, oid uint32, data []byte) (any, error) {
	dt, ok := m.TypeForOID(oid)
	if !ok {
		return string(data), nil
	}

	val, err := dt.Codec.DecodeValue(m, oid, pgtype.TextFormatCode, data)
	if err != nil {
		return nil, err
	}

	if b, ok := val.([16]byte); ok && oid == pgtype.UUIDOID {
		return uuidString(b), nil
	}

	if im, ok := val.(pgtype.InfinityModifier); ok {
		return infiniteTime(im), nil
	}

	return val, nil
}

// infiniteTime is a date, timestamp or timestamptz infinity or -infinity. It
// prints as PostgreSQL spells it and serializes as null, which JSON and YAML
// have no better spelling for.
type infiniteTime pgtype.InfinityModifier

// String renders the value as "infinity" or "-infinity".
func (t infiniteTime) String() string { return pgtype.InfinityModifier(t).String() }

// MarshalJSON encodes the value as JSON null.
func (infiniteTime) MarshalJSON() ([]byte, error) { return []byte("null"), nil }

// MarshalYAML encodes the value as YAML null.
func (infiniteTime) MarshalYAML() (any, error) { return nil, nil }

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

// emptyMessages is the shared result of dump() when nothing is buffered. It
// must never be written to: every caller holding it shares the same map.
var emptyMessages = map[string]string{}

// append stores a message under the given name, overwriting any prior value.
func (c *postgresMessageContext) append(name, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages[name] = value
}

// reset drops the buffered messages.
func (c *postgresMessageContext) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages = map[string]string{}
}

// dump returns a copy of the buffered messages without clearing the store, or
// a shared empty map when nothing is buffered. The result is read-only.
func (c *postgresMessageContext) dump() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.messages) == 0 {
		return emptyMessages
	}

	messages := make(map[string]string, len(c.messages))
	maps.Copy(messages, c.messages)

	return messages
}

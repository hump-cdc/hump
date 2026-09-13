package postgres

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/goleak"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestNewWalOpensReplicationSession(t *testing.T) {
	server := usersServer(t)

	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	if w.dbName != "db" {
		t.Errorf("dbName = %q, want %q", w.dbName, "db")
	}

	if w.session == nil {
		t.Error("session = nil, want an open replication session")
	}

	want := []string{"proto_version '2'", "publication_names 'hump_pub'", "messages 'true'"}
	if !slices.Equal(w.pluginArguments, want) {
		t.Errorf("pluginArguments = %q, want %q", w.pluginArguments, want)
	}
}

func TestNewWalFailsWhenSessionCannotBeAcquired(t *testing.T) {
	conn, err := newConnection(fmt.Sprintf("postgres://u:p@%s/db?sslmode=disable", unusedAddress(t)), withMaxAttempts(1))
	if err != nil {
		t.Fatalf("newConnection() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	_, err = newWal(context.Background(), conn, "hump_pub", "hump_slot", nil, newFakeHealth(), newTestMetrics())
	if err == nil || !errors.Is(err, database.ErrWal) {
		t.Fatalf("newWal() error = %v, want it to wrap ErrWal", err)
	}
}

func TestStartFailsWhenReplicationCannotStart(t *testing.T) {
	server := usersServer(t)
	server.failReplicationStart("42704", "replication slot does not exist")

	w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	err := w.Start(context.Background(), failingHandler(t))
	if err == nil || !strings.Contains(err.Error(), "StartReplication failed") {
		t.Fatalf("Start() error = %v, want it to mention a failed StartReplication", err)
	}

	if !errors.Is(err, database.ErrWal) {
		t.Errorf("Start() error does not wrap ErrWal: %v", err)
	}

	if got := health.healthyCount(); got != 0 {
		t.Errorf("MarkHealthy() called %d times, want 0", got)
	}
}

func TestStartStopsWhenContextIsCancelled(t *testing.T) {
	server := usersServer(t)
	w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := runStart(ctx, w, failingHandler(t))
	health.waitForHealthy(t)
	cancel()

	err := awaitStart(t, errs)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start() error = %v, want context.Canceled", err)
	}

	if errors.Is(err, database.ErrWal) {
		t.Errorf("Start() error = %v, want a shutdown rather than a WAL failure", err)
	}

	if failures := health.failures(); len(failures) != 0 {
		t.Errorf("MarkUnhealthy() called %d times, want 0", len(failures))
	}

	server.waitForStandbyUpdates(t, 1)
}

func TestStartFinishesHandlerWhenContextIsCancelled(t *testing.T) {
	server := usersServer(t)
	w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)), configuredTable("public", "users"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entered := make(chan struct{})
	release := make(chan struct{})
	var handlerCtxErr error
	handler := func(ctx context.Context, _ time.Time, _ domain.OperationType, _ domain.DbName, _ domain.TableSchema, _ domain.TableName, _ domain.Headers, _ domain.DbRecord, _ domain.DbRecord) error {
		close(entered)
		<-release
		handlerCtxErr = ctx.Err()

		return nil
	}

	errs := runStart(ctx, w, handler)

	insert := insertMessage(1, textColumn("7"))
	server.stream(xLogData(0x10, 0x1000, relationMessage(1, "public", "users", idColumn())))
	server.stream(xLogData(0x100, 0x1000, insert))

	<-entered
	cancel()
	close(release)

	err := awaitStart(t, errs)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start() error = %v, want context.Canceled", err)
	}

	if errors.Is(err, database.ErrWal) {
		t.Errorf("Start() error = %v, want a shutdown rather than a WAL failure", err)
	}

	if handlerCtxErr != nil {
		t.Errorf("handler context error = %v, want nil: an in-flight change must finish on shutdown", handlerCtxErr)
	}

	if failures := health.failures(); len(failures) != 0 {
		t.Errorf("MarkUnhealthy() called %d times, want 0", len(failures))
	}

	server.waitForStandbyPosition(t, pglogrepl.LSN(0x100+len(insert)))
}

func TestStartReportsHandlerFailureWhenContextIsCancelled(t *testing.T) {
	server := usersServer(t)
	w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)), configuredTable("public", "users"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wantErr := errors.New("sink unavailable")
	entered := make(chan struct{})
	release := make(chan struct{})
	handler := func(context.Context, time.Time, domain.OperationType, domain.DbName, domain.TableSchema, domain.TableName, domain.Headers, domain.DbRecord, domain.DbRecord) error {
		close(entered)
		<-release

		return wantErr
	}

	errs := runStart(ctx, w, handler)

	relation := relationMessage(1, "public", "users", idColumn())
	server.stream(xLogData(0x10, 0x1000, relation))
	server.stream(xLogData(0x100, 0x1000, insertMessage(1, textColumn("7"))))

	<-entered
	cancel()
	close(release)

	err := awaitStart(t, errs)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Start() error = %v, want %v", err, wantErr)
	}

	if errors.Is(err, context.Canceled) {
		t.Errorf("Start() error = %v, want the handler failure rather than a shutdown", err)
	}

	if failures := health.failures(); len(failures) != 1 {
		t.Errorf("MarkUnhealthy() called %d times, want 1", len(failures))
	}

	last := pglogrepl.LSN(0x10 + len(relation))
	for _, update := range server.waitForStandbyUpdates(t, 1) {
		if update.WALWritePosition > last {
			t.Errorf("standby update reports %s, want at most %s: the failed change must not be confirmed", update.WALWritePosition, last)
		}
	}
}

func TestStartKeepsWaitingAfterReceiveTimeout(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := w.Start(ctx, failingHandler(t))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Start() error = %v, want context.DeadlineExceeded", err)
	}

	if strings.Contains(err.Error(), "receive message failed") {
		t.Errorf("Start() error = %v, want the receive timeout to be retried rather than reported", err)
	}

	server.waitForStandbyUpdates(t, 1)
}

func TestStartAdvancesWalPositionAcrossChanges(t *testing.T) {
	server := usersServer(t)
	w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)), configuredTable("public", "users"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := newHandlerRecorder(nil)
	errs := runStart(ctx, w, recorder.handle)

	insert := insertMessage(1, textColumn("7"))

	server.stream(keepalive(0x1000, false))
	server.stream(xLogData(0x10, 0x1000, relationMessage(1, "public", "users", idColumn())))
	server.stream(xLogData(0x100, 0x1000, insert))
	server.stream(keepalive(0x1000, true))

	updates := server.waitForStandbyUpdates(t, 1)
	if len(updates) != 1 {
		t.Fatalf("standby updates = %d, want 1: a keepalive that asks for no reply must not flush", len(updates))
	}

	want := pglogrepl.LSN(0x100 + len(insert))
	if updates[0].WALWritePosition != want || updates[0].WALFlushPosition != want {
		t.Errorf("standby update = write %s, flush %s, want %s for both", updates[0].WALWritePosition, updates[0].WALFlushPosition, want)
	}

	cancel()
	awaitStart(t, errs)

	if got := len(recorder.recorded()); got != 1 {
		t.Errorf("handler called %d times, want 1", got)
	}

	if got := health.healthyCount(); got != 1 {
		t.Errorf("MarkHealthy() called %d times, want 1", got)
	}
}

func TestStartKeepsReceivingAfterReplyRequestedFlush(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)), configuredTable("public", "users"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := newHandlerRecorder(nil)
	errs := runStart(ctx, w, recorder.handle)

	first := insertMessage(1, textColumn("7"))
	second := insertMessage(1, textColumn("8"))

	server.stream(xLogData(0x10, 0x1000, relationMessage(1, "public", "users", idColumn())))
	server.stream(xLogData(0x100, 0x1000, first))
	server.stream(keepalive(0x1000, true))

	server.waitForStandbyPosition(t, pglogrepl.LSN(0x100+len(first)))

	server.stream(xLogData(0x200, 0x1000, second))
	server.stream(keepalive(0x1000, true))

	server.waitForStandbyPosition(t, pglogrepl.LSN(0x200+len(second)))

	cancel()
	awaitStart(t, errs)

	if got := len(recorder.recorded()); got != 2 {
		t.Errorf("handler called %d times, want 2: the receive context must survive a flush", got)
	}
}

func TestStartKeepsReceivingAcrossStandbyTicks(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)), configuredTable("public", "users"))
	w.standbyInterval = testStandbyInterval

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := newHandlerRecorder(nil)
	errs := runStart(ctx, w, recorder.handle)

	server.waitForStandbyUpdates(t, 3)

	insert := insertMessage(1, textColumn("7"))

	server.stream(xLogData(0x10, 0x1000, relationMessage(1, "public", "users", idColumn())))
	server.stream(xLogData(0x100, 0x1000, insert))

	server.waitForStandbyPosition(t, pglogrepl.LSN(0x100+len(insert)))

	cancel()
	awaitStart(t, errs)

	if got := len(recorder.recorded()); got != 1 {
		t.Errorf("handler called %d times, want 1: the receive context must survive a standby tick", got)
	}
}

func TestStartUpdatesReplicationLagOnKeepalive(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := runStart(ctx, w, failingHandler(t))

	relation := relationMessage(1, "public", "users", idColumn())
	server.stream(xLogData(0x10, 0x1000, relation))
	server.stream(keepalive(0x2000, true))
	server.waitForStandbyUpdates(t, 1)

	cancel()
	awaitStart(t, errs)

	want := float64(0x2000 - (0x10 + len(relation)))
	if got := testutil.ToFloat64(w.metrics.replicationLag); got != want {
		t.Errorf("replicationLag = %v, want %v: a keepalive is the only lag update an idle stream gets", got, want)
	}
}

func TestStartLeavesReplicationLagUnsetBeforeFirstRecord(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := runStart(ctx, w, failingHandler(t))

	server.stream(keepalive(0x1000, true))
	updates := server.waitForStandbyUpdates(t, 1)

	cancel()
	awaitStart(t, errs)

	// The reply is what wal_sender_timeout waits for, so it goes out even with
	// nothing to confirm; only the lag it cannot compute yet is withheld.
	if updates[0].WALFlushPosition != 0 {
		t.Errorf("standby update flush = %s, want 0/0 until the first record is decoded", updates[0].WALFlushPosition)
	}

	if got := testutil.ToFloat64(w.metrics.replicationLag); got != 0 {
		t.Errorf("replicationLag = %v, want 0: the server LSN alone is not a lag", got)
	}
}

func TestStartKeepsReplicationLagOnStreamFailure(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := runStart(ctx, w, failingHandler(t))

	relation := relationMessage(1, "public", "users", idColumn())
	server.stream(xLogData(0x10, 0x1000, relation))
	server.stream(keepalive(0x2000, true))
	server.waitForStandbyUpdates(t, 1)

	server.stream(copyFrame(pglogrepl.PrimaryKeepaliveMessageByteID, 0, 0))
	if err := awaitStart(t, errs); err == nil {
		t.Fatal("Start() error = nil, want the malformed keepalive to stop the stream")
	}

	want := float64(0x2000 - (0x10 + len(relation)))
	if got := testutil.ToFloat64(w.metrics.replicationLag); got != want {
		t.Errorf("replicationLag = %v, want %v: an unparseable message carries no WAL position to report", got, want)
	}
}

func TestStartIgnoresUnexpectedMessage(t *testing.T) {
	server := usersServer(t)
	w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := runStart(ctx, w, failingHandler(t))

	server.stream(&pgproto3.ParameterStatus{Name: "application_name", Value: "hump"})
	server.stream(keepalive(0x100, true))

	server.waitForStandbyUpdates(t, 1)

	cancel()
	awaitStart(t, errs)
}

func TestStartStopsOnStreamFailure(t *testing.T) {
	tests := []struct {
		name    string
		frame   pgproto3.BackendMessage
		wantErr string
		wantIs  error
	}{
		{"malformed keepalive", copyFrame(pglogrepl.PrimaryKeepaliveMessageByteID, 0, 0), "ParsePrimaryKeepaliveMessage failed", nil},
		{"malformed xlog data", copyFrame(pglogrepl.XLogDataByteID, 0, 0), "ParseXLogData failed", nil},
		{"server error", &pgproto3.ErrorResponse{Severity: "ERROR", Code: "58000", Message: "WAL sender failed"}, "received Postgres WAL error", nil},
		{"unknown relation", xLogData(0x10, 0x100, insertMessage(9, textColumn("7"))), "processMessage failed", nil},
		{"empty copy data", copyFrame(), "received empty CopyData frame", errEmptyCopyData},
		{"empty wal record", xLogData(0x10, 0x100, nil), "processMessage failed", errEmptyWalRecord},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := usersServer(t)
			w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			errs := runStart(ctx, w, failingHandler(t))
			server.stream(tt.frame)

			err := awaitStart(t, errs)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Start() error = %v, want it to mention %q", err, tt.wantErr)
			}

			if !errors.Is(err, database.ErrWal) {
				t.Errorf("Start() error does not wrap ErrWal: %v", err)
			}

			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("Start() error = %v, want it to wrap %v", err, tt.wantIs)
			}

			if failures := health.failures(); len(failures) != 1 {
				t.Errorf("MarkUnhealthy() called %d times, want 1", len(failures))
			}
		})
	}
}

func TestStartStopsWhenStandbyUpdateFails(t *testing.T) {
	server := usersServer(t)

	conn := newTestConnection(t, server, withReplication(true))
	conn.connectionConfig.DialFunc = brokenWriteDialer(2)

	w, health := newReplicationWal(t, server, conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := runStart(ctx, w, failingHandler(t))
	server.stream(keepalive(0x100, true))

	err := awaitStart(t, errs)
	if err == nil || !strings.Contains(err.Error(), "send status failed") {
		t.Fatalf("Start() error = %v, want it to mention a failed status update", err)
	}

	if !errors.Is(err, database.ErrWal) {
		t.Errorf("Start() error does not wrap ErrWal: %v", err)
	}

	if failures := health.failures(); len(failures) != 1 {
		t.Errorf("MarkUnhealthy() called %d times, want 1", len(failures))
	}
}

func TestFlushReportsUnhealthyWhenStatusCannotBeSent(t *testing.T) {
	server := usersServer(t)
	w, health := newReplicationWal(t, server, newTestConnection(t, server, withReplication(true)))

	if err := w.session.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	err := w.flush(context.Background(), 42)
	if err == nil || !strings.Contains(err.Error(), "send status failed") {
		t.Fatalf("flush() error = %v, want it to mention a failed status update", err)
	}

	if !errors.Is(err, database.ErrWal) {
		t.Errorf("flush() error does not wrap ErrWal: %v", err)
	}

	if failures := health.failures(); len(failures) != 1 {
		t.Errorf("MarkUnhealthy() called %d times, want 1", len(failures))
	}
}

func TestStopClosesConnectionAndReportsUnhealthy(t *testing.T) {
	server := usersServer(t)
	conn := newTestConnection(t, server, withReplication(true))

	w, health := newReplicationWal(t, server, conn)

	if err := w.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if session := heldSession(conn); session != nil {
		t.Error("Stop() left a session cached, want it closed")
	}

	failures := health.failures()
	if len(failures) != 1 || !errors.Is(failures[0], database.ErrWalClosed) {
		t.Errorf("MarkUnhealthy() got %v, want a single ErrWalClosed", failures)
	}
}

func TestProcessMessageRejectsUnparseableMessage(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))

	err := w.processMessage(context.Background(), []byte{'z'}, time.Now(), 0, failingHandler(t))
	if err == nil || !strings.Contains(err.Error(), "Parse logical replication message") {
		t.Fatalf("processMessage() error = %v, want it to mention a failed parse", err)
	}
}

func TestProcessMessageRejectsEmptyMessage(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))

	err := w.processMessage(context.Background(), nil, time.Now(), 0, failingHandler(t))
	if !errors.Is(err, errEmptyWalRecord) {
		t.Fatalf("processMessage() error = %v, want errEmptyWalRecord", err)
	}
}

func TestProcessMessageCachesRelation(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))

	registerRelation(t, w, 1, "public", "users", idColumn())

	relation, ok := w.relations[1]
	if !ok {
		t.Fatal("relation 1 is not cached")
	}

	if relation.Namespace != "public" || relation.RelationName != "users" {
		t.Errorf("cached relation = %s.%s, want public.users", relation.Namespace, relation.RelationName)
	}
}

func TestProcessMessageIgnoresTransactionStart(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))

	if err := w.processMessage(context.Background(), beginMessage(), time.Now(), 0, failingHandler(t)); err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}
}

func TestProcessMessageBuffersEmittedMessage(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))

	if err := w.processMessage(context.Background(), logicalDecodingMessage(true, "trace", "abc"), time.Now(), 0, failingHandler(t)); err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if got := w.messageContext.dump(); !maps.Equal(got, map[string]string{"trace": "abc"}) {
		t.Errorf("buffered messages = %v, want {trace: abc}", got)
	}
}

func TestProcessMessageSkipsUnusableEmittedMessage(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
	}{
		{"not transactional", logicalDecodingMessage(false, "trace", "abc")},
		{"without prefix", logicalDecodingMessage(true, "", "abc")},
		{"without content", logicalDecodingMessage(true, "trace", "")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t))

			if err := w.processMessage(context.Background(), tt.message, time.Now(), 0, failingHandler(t)); err != nil {
				t.Fatalf("processMessage() error = %v", err)
			}

			if got := w.messageContext.dump(); len(got) != 0 {
				t.Errorf("buffered messages = %v, want none", got)
			}
		})
	}
}

func TestProcessMessageClearsEmittedMessagesOnCommit(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))

	if err := w.processMessage(context.Background(), logicalDecodingMessage(true, "trace", "abc"), time.Now(), 0, failingHandler(t)); err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if err := w.processMessage(context.Background(), commitMessage(), time.Now(), 0, failingHandler(t)); err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if got := w.messageContext.dump(); len(got) != 0 {
		t.Errorf("buffered messages = %v, want none after a commit", got)
	}
}

func TestProcessMessageDispatchesRowChanges(t *testing.T) {
	tests := []struct {
		name      string
		message   []byte
		operation domain.OperationType
		wantPrev  domain.DbRecord
		wantNext  domain.DbRecord
	}{
		{
			name:      "insert",
			message:   insertMessage(1, textColumn("7")),
			operation: domain.OperationTypeInsert,
			wantNext:  domain.DbRecord{"id": int32(7)},
		},
		{
			name:      "update",
			message:   updateMessage(1, []tupleColumn{textColumn("6")}, []tupleColumn{textColumn("7")}),
			operation: domain.OperationTypeUpdate,
			wantPrev:  domain.DbRecord{"id": int32(6)},
			wantNext:  domain.DbRecord{"id": int32(7)},
		},
		{
			name:      "delete",
			message:   deleteMessage(1, textColumn("6")),
			operation: domain.OperationTypeDelete,
			wantPrev:  domain.DbRecord{"id": int32(6)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
			registerRelation(t, w, 1, "public", "users", idColumn())
			w.messageContext.append("trace", "abc")

			recorder := newHandlerRecorder(nil)

			timestamp := time.Unix(1700000000, 0).UTC()
			if err := w.processMessage(context.Background(), tt.message, timestamp, 0, recorder.handle); err != nil {
				t.Fatalf("processMessage() error = %v", err)
			}

			calls := recorder.recorded()
			if len(calls) != 1 {
				t.Fatalf("handler called %d times, want 1", len(calls))
			}

			call := calls[0]
			if call.operation != tt.operation {
				t.Errorf("operation = %q, want %q", call.operation, tt.operation)
			}

			if call.dbName != "db" || call.schema != "public" || call.table != "users" {
				t.Errorf("target = %s %s.%s, want db public.users", call.dbName, call.schema, call.table)
			}

			if !call.timestamp.Equal(timestamp) {
				t.Errorf("timestamp = %s, want %s", call.timestamp, timestamp)
			}

			if !maps.Equal(call.headers, domain.Headers{"trace": "abc"}) {
				t.Errorf("headers = %v, want {trace: abc}", call.headers)
			}

			if !maps.Equal(call.prev, tt.wantPrev) {
				t.Errorf("prev = %v, want %v", call.prev, tt.wantPrev)
			}

			if !maps.Equal(call.next, tt.wantNext) {
				t.Errorf("next = %v, want %v", call.next, tt.wantNext)
			}
		})
	}
}

func TestProcessMessageReportsKeyOnlyOldTupleAsUnavailable(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
	registerRelation(t, w, 1, "public", "users", idColumn(), blobColumn())

	recorder := newHandlerRecorder(nil)
	message := keyUpdateMessage(1,
		[]tupleColumn{textColumn("6"), nullColumn()},
		[]tupleColumn{textColumn("7"), toastColumn()},
	)

	if err := w.processMessage(context.Background(), message, time.Now(), 0, recorder.handle); err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	calls := recorder.recorded()
	if len(calls) != 1 {
		t.Fatalf("handler called %d times, want 1", len(calls))
	}

	wantPrev := domain.DbRecord{"id": int32(6), "blob": domain.Unavailable}
	wantNext := domain.DbRecord{"id": int32(7), "blob": domain.Unavailable}

	if !maps.Equal(calls[0].prev, wantPrev) {
		t.Errorf("prev = %v, want %v", calls[0].prev, wantPrev)
	}

	if !maps.Equal(calls[0].next, wantNext) {
		t.Errorf("next = %v, want %v", calls[0].next, wantNext)
	}
}

func TestProcessMessageDispatchesTruncateForEveryRelation(t *testing.T) {
	server := twoTableServer(t)
	w, _ := newTestWal(t, server, configuredTable("public", "users"), configuredTable("public", "orders"))
	registerRelation(t, w, 1, "public", "users", idColumn())
	registerRelation(t, w, 2, "public", "orders", idColumn())

	recorder := newHandlerRecorder(nil)

	if err := w.processMessage(context.Background(), truncateMessage(1, 2), time.Now(), 0, recorder.handle); err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	calls := recorder.recorded()
	if len(calls) != 2 {
		t.Fatalf("handler called %d times, want 2", len(calls))
	}

	for _, call := range calls {
		if call.operation != domain.OperationTypeTruncate {
			t.Errorf("operation = %q, want truncate", call.operation)
		}

		if call.prev != nil || call.next != nil {
			t.Errorf("truncate carried rows prev = %v, next = %v, want neither", call.prev, call.next)
		}
	}

	if got := []domain.TableName{calls[0].table, calls[1].table}; !slices.Equal(got, []domain.TableName{"users", "orders"}) {
		t.Errorf("truncated tables = %v, want [users orders]", got)
	}
}

func TestProcessMessageJoinsTruncateFailures(t *testing.T) {
	server := twoTableServer(t)
	w, _ := newTestWal(t, server, configuredTable("public", "users"), configuredTable("public", "orders"))
	registerRelation(t, w, 1, "public", "users", idColumn())
	registerRelation(t, w, 2, "public", "orders", idColumn())

	failure := errors.New("sink unavailable")
	recorder := newHandlerRecorder(failure)

	err := w.processMessage(context.Background(), truncateMessage(1, 2), time.Now(), 0, recorder.handle)
	if !errors.Is(err, failure) {
		t.Fatalf("processMessage() error = %v, want it to wrap %v", err, failure)
	}

	if got := len(recorder.recorded()); got != 2 {
		t.Errorf("handler called %d times, want 2: a failure must not stop the remaining relations", got)
	}
}

func TestProcessMessageRejectsUnknownRelation(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
	}{
		{"insert", insertMessage(9, textColumn("7"))},
		{"update", updateMessage(9, nil, []tupleColumn{textColumn("7")})},
		{"delete", deleteMessage(9, textColumn("7"))},
		{"truncate", truncateMessage(9)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
			registerRelation(t, w, 1, "public", "users", idColumn())

			err := w.processMessage(context.Background(), tt.message, time.Now(), 0, failingHandler(t))
			if err == nil || !strings.Contains(err.Error(), "unknown relation ID: 9") {
				t.Fatalf("processMessage() error = %v, want it to mention the unknown relation", err)
			}

			if !errors.Is(err, database.ErrWal) {
				t.Errorf("processMessage() error does not wrap ErrWal: %v", err)
			}
		})
	}
}

func TestProcessMessageReportsDecodeFailures(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
		wantErr string
	}{
		{"insert", insertMessage(1, textColumn("seven")), "decode insert tuples for public.users"},
		{"update old row", updateMessage(1, []tupleColumn{textColumn("six")}, []tupleColumn{textColumn("7")}), "decode update tuples for public.users"},
		{"update new row", updateMessage(1, []tupleColumn{textColumn("6")}, []tupleColumn{textColumn("seven")}), "decode update tuples for public.users"},
		{"delete", deleteMessage(1, textColumn("six")), "decode delete tuples for public.users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
			registerRelation(t, w, 1, "public", "users", idColumn())

			err := w.processMessage(context.Background(), tt.message, time.Now(), 0, failingHandler(t))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("processMessage() error = %v, want it to mention %q", err, tt.wantErr)
			}

			if !strings.Contains(err.Error(), `decode column "id" (oid 23)`) {
				t.Errorf("processMessage() error = %v, want it to name the column that failed", err)
			}
		})
	}
}

func TestProcessMessageReportsResolveFailures(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
	}{
		{"insert", insertMessage(1, textColumn("7"))},
		{"update", updateMessage(1, nil, []tupleColumn{textColumn("7")})},
		{"delete", deleteMessage(1, textColumn("7"))},
		{"truncate", truncateMessage(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(failingOn(tableLayoutResponder(false), "unnest(", serverError("42501", "permission denied")))

			w, _ := newTestWal(t, server, configuredTable("public", "users"))
			registerRelation(t, w, 1, "public", "users", idColumn())

			err := w.processMessage(context.Background(), tt.message, time.Now(), 0, failingHandler(t))
			if err == nil || !strings.Contains(err.Error(), "failed to resolve table configuration for public.users") {
				t.Fatalf("processMessage() error = %v, want it to mention the unresolved table", err)
			}
		})
	}
}

func TestProcessMessageSkipsIgnoredTables(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
	}{
		{"insert", insertMessage(2, textColumn("seven"))},
		{"update", updateMessage(2, nil, []tupleColumn{textColumn("seven")})},
		{"delete", deleteMessage(2, textColumn("seven"))},
		{"truncate", truncateMessage(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
			registerRelation(t, w, 2, "public", "audit", idColumn())

			if err := w.processMessage(context.Background(), tt.message, time.Now(), 0, failingHandler(t)); err != nil {
				t.Fatalf("processMessage() error = %v, want an unconfigured table to be skipped", err)
			}
		})
	}
}

func TestProcessMessageReturnsHandlerFailure(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
	}{
		{"insert", insertMessage(1, textColumn("7"))},
		{"update", updateMessage(1, nil, []tupleColumn{textColumn("7")})},
		{"delete", deleteMessage(1, textColumn("7"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
			registerRelation(t, w, 1, "public", "users", idColumn())

			failure := errors.New("sink unavailable")
			recorder := newHandlerRecorder(failure)

			if err := w.processMessage(context.Background(), tt.message, time.Now(), 0, recorder.handle); !errors.Is(err, failure) {
				t.Fatalf("processMessage() error = %v, want %v", err, failure)
			}
		})
	}
}

func TestResolveTableAnswersCachedLookupsWithoutGoroutines(t *testing.T) {
	server := usersServer(t)
	w, _ := newTestWal(t, server, configuredTable("public", "users"))

	table := domain.TableIdentifier{Schema: "public", Name: "users"}
	if _, err := w.resolveTable(context.Background(), table, 0); err != nil {
		t.Fatalf("resolveTable() error = %v", err)
	}

	leaks := goleak.IgnoreCurrent()

	for range 100 {
		if _, err := w.resolveTable(context.Background(), table, 0); err != nil {
			t.Fatalf("resolveTable() error = %v", err)
		}
	}

	if err := goleak.Find(leaks); err != nil {
		t.Errorf("resolveTable() leaked goroutines on the cached path: %v", err)
	}
}

func TestResolveTableKeepsTheStreamAliveDuringIntrospection(t *testing.T) {
	server := usersServer(t)

	// Blocking the responder would deadlock the fake server, which holds its
	// mutex across the script; a slow ensureReplicaIdentity stalls introspection
	// the same way an ALTER waiting on a lock does.
	released := make(chan struct{})
	ensure := func(context.Context, *connection, []domain.TableIdentifier) error {
		<-released

		return nil
	}

	conn := newTestConnection(t, server, withReplication(true))
	w, _ := newReplicationWal(t, server, conn, configuredTable("public", "users"))
	w.introspector = newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, ensure)
	w.standbyInterval = testStandbyInterval

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pglogrepl.StartReplication(ctx, w.session, w.slotName, 0, pglogrepl.StartReplicationOptions{PluginArgs: w.pluginArguments}); err != nil {
		t.Fatalf("StartReplication() error = %v", err)
	}

	position := pglogrepl.LSN(4242)
	done := make(chan error, 1)
	go func() {
		_, err := w.resolveTable(ctx, domain.TableIdentifier{Schema: "public", Name: "users"}, position)
		done <- err
	}()

	updates := server.waitForStandbyUpdates(t, 2)
	for i, update := range updates {
		if update.WALWritePosition != position {
			t.Errorf("standby update %d position = %s, want the last processed %s", i, update.WALWritePosition, position)
		}
	}

	close(released)
	if err := <-done; err != nil {
		t.Fatalf("resolveTable() error = %v", err)
	}
}

func TestResolveTableStopsTheKeepaliveBeforeReturning(t *testing.T) {
	server := usersServer(t)

	released := make(chan struct{})
	ensure := func(context.Context, *connection, []domain.TableIdentifier) error {
		<-released

		return nil
	}

	conn := newTestConnection(t, server, withReplication(true))
	w, _ := newReplicationWal(t, server, conn, configuredTable("public", "users"))
	w.introspector = newTestIntrospector(t, server, []config.Table{configuredTable("public", "users")}, ensure)
	w.standbyInterval = testStandbyInterval

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pglogrepl.StartReplication(ctx, w.session, w.slotName, 0, pglogrepl.StartReplicationOptions{PluginArgs: w.pluginArguments}); err != nil {
		t.Fatalf("StartReplication() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := w.resolveTable(ctx, domain.TableIdentifier{Schema: "public", Name: "users"}, 0)
		done <- err
	}()

	// Release only once the keepalive is demonstrably running, so the assertion
	// below is about stop() waiting for it rather than about it never starting.
	server.waitForStandbyUpdates(t, 1)
	close(released)

	if err := <-done; err != nil {
		t.Fatalf("resolveTable() error = %v", err)
	}

	before := len(server.standbyUpdates())
	time.Sleep(5 * testStandbyInterval)
	if after := len(server.standbyUpdates()); after != before {
		t.Errorf("standby updates grew from %d to %d after resolveTable returned", before, after)
	}
}

func TestDecodeRowWithoutTuples(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn())

	prev, next, err := w.decodeRow(nil, pglogrepl.UpdateMessageTupleTypeNone, nil, w.relations[1])
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	if prev != nil || next != nil {
		t.Errorf("decodeRow() = %v, %v, want nil for both", prev, next)
	}
}

func TestDecodeRowKeepsNullColumns(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn())

	prev, next, err := w.decodeRow(tupleData(nullColumn()), pglogrepl.UpdateMessageTupleTypeOld, tupleData(nullColumn()), w.relations[1])
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	for name, got := range map[string]domain.DbRecord{"prev": prev, "next": next} {
		value, ok := got["id"]
		if !ok || value != nil {
			t.Errorf("%s = %v, want id present and nil", name, got)
		}
	}
}

func TestDecodeRowCarriesUnchangedToastForward(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn(), blobColumn())

	prev, next, err := w.decodeRow(
		tupleData(textColumn("1"), textColumn("stored")),
		pglogrepl.UpdateMessageTupleTypeOld,
		tupleData(textColumn("1"), toastColumn()),
		w.relations[1],
	)
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	if prev["blob"] != "stored" {
		t.Errorf(`prev["blob"] = %#v, want "stored"`, prev["blob"])
	}

	if next["blob"] != "stored" {
		t.Errorf(`next["blob"] = %#v, want the previous value carried forward`, next["blob"])
	}
}

func TestDecodeRowReportsUnchangedToastWithNothingToCarryAsUnavailable(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn(), blobColumn())

	prev, next, err := w.decodeRow(
		tupleData(textColumn("1"), toastColumn()),
		pglogrepl.UpdateMessageTupleTypeOld,
		tupleData(textColumn("1"), toastColumn()),
		w.relations[1],
	)
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	for name, got := range map[string]domain.DbRecord{"prev": prev, "next": next} {
		if got["blob"] != domain.Unavailable {
			t.Errorf(`%s["blob"] = %#v, want domain.Unavailable`, name, got["blob"])
		}
	}
}

func TestDecodeRowReportsUnchangedToastWithoutOldTupleAsUnavailable(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn(), blobColumn())

	prev, next, err := w.decodeRow(
		nil,
		pglogrepl.UpdateMessageTupleTypeNone,
		tupleData(textColumn("1"), toastColumn()),
		w.relations[1],
	)
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	if prev != nil {
		t.Errorf("prev = %v, want nil", prev)
	}

	if next["blob"] != domain.Unavailable {
		t.Errorf(`next["blob"] = %#v, want domain.Unavailable`, next["blob"])
	}
}

func TestDecodeRowReportsNonKeyColumnsOfKeyOnlyTupleAsUnavailable(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn(), blobColumn())

	prev, next, err := w.decodeRow(
		tupleData(textColumn("1"), nullColumn()),
		pglogrepl.UpdateMessageTupleTypeKey,
		tupleData(textColumn("2"), toastColumn()),
		w.relations[1],
	)
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	if prev["id"] != int32(1) {
		t.Errorf(`prev["id"] = %#v, want the key column decoded as int32(1)`, prev["id"])
	}

	for name, got := range map[string]domain.DbRecord{"prev": prev, "next": next} {
		if got["blob"] != domain.Unavailable {
			t.Errorf(`%s["blob"] = %#v, want domain.Unavailable rather than a fabricated NULL`, name, got["blob"])
		}
	}
}

func TestDecodeRowKeepsKeyColumnsOfKeyOnlyTuple(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn(), relationColumn{name: "email", oid: pgtype.TextOID, key: true})

	prev, _, err := w.decodeRow(
		tupleData(textColumn("1"), textColumn("a@b.c")),
		pglogrepl.UpdateMessageTupleTypeKey,
		nil,
		w.relations[1],
	)
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	if prev["id"] != int32(1) || prev["email"] != "a@b.c" {
		t.Errorf("prev = %#v, want every replica identity column decoded", prev)
	}
}

func TestDecodeRowKeepsNullsOfFullOldTuple(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t))
	registerRelation(t, w, 1, "public", "users", idColumn(), blobColumn())

	prev, _, err := w.decodeRow(
		tupleData(textColumn("1"), nullColumn()),
		pglogrepl.UpdateMessageTupleTypeOld,
		nil,
		w.relations[1],
	)
	if err != nil {
		t.Fatalf("decodeRow() error = %v", err)
	}

	value, ok := prev["blob"]
	if !ok || value != nil {
		t.Errorf(`prev["blob"] = %#v, want a genuine NULL`, prev["blob"])
	}
}

func TestDecodeRowRejectsUnexpectedDataType(t *testing.T) {
	tests := []struct {
		name    string
		prevRow *pglogrepl.TupleData
		nextRow *pglogrepl.TupleData
	}{
		{"old tuple", tupleData(tupleColumn{dataType: 'b', data: "7"}), nil},
		{"new tuple", nil, tupleData(tupleColumn{dataType: 'b', data: "7"})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t))
			registerRelation(t, w, 1, "public", "users", idColumn())

			_, _, err := w.decodeRow(tt.prevRow, pglogrepl.UpdateMessageTupleTypeOld, tt.nextRow, w.relations[1])
			if err == nil || !strings.Contains(err.Error(), `unexpected tuple data type 'b' for column "id"`) {
				t.Fatalf("decodeRow() error = %v, want it to name the unexpected data type and column", err)
			}
		})
	}
}

func TestDecodeRowRejectsColumnCountMismatch(t *testing.T) {
	tests := []struct {
		name    string
		prevRow *pglogrepl.TupleData
		nextRow *pglogrepl.TupleData
		wantErr string
	}{
		{"old tuple too long", tupleData(textColumn("1"), textColumn("2")), nil, "old tuple has 2 columns, relation has 1"},
		{"old tuple too short", tupleData(), nil, "old tuple has 0 columns, relation has 1"},
		{"new tuple too long", nil, tupleData(textColumn("1"), textColumn("2")), "new tuple has 2 columns, relation has 1"},
		{"new tuple too short", nil, tupleData(), "new tuple has 0 columns, relation has 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t))
			registerRelation(t, w, 1, "public", "users", idColumn())

			_, _, err := w.decodeRow(tt.prevRow, pglogrepl.UpdateMessageTupleTypeOld, tt.nextRow, w.relations[1])
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("decodeRow() error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeRowText(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		dataType uint32
		want     any
		wantErr  bool
	}{
		{name: "known type", data: "7", dataType: pgtype.Int4OID, want: int32(7)},
		{name: "unknown type", data: "whatever", dataType: 999999, want: "whatever"},
		{name: "uuid", data: "0f8fad5b-d9cb-469f-a165-70867728950e", dataType: pgtype.UUIDOID, want: "0f8fad5b-d9cb-469f-a165-70867728950e"},
		{name: "undecodable", data: "seven", dataType: pgtype.Int4OID, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := newTestWal(t, usersServer(t))

			got, err := w.decodeRowText([]byte(tt.data), tt.dataType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("decodeRowText() = %#v, want an error", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("decodeRowText() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("decodeRowText() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMessageContextResetClearsBufferedMessages(t *testing.T) {
	messageContext := &postgresMessageContext{messages: map[string]string{}}
	messageContext.append("trace", "abc")
	messageContext.append("trace", "def")

	if got := messageContext.dump(); !maps.Equal(got, map[string]string{"trace": "def"}) {
		t.Errorf("dump() = %v, want the latest value under each name", got)
	}

	messageContext.reset()

	if got := messageContext.dump(); len(got) != 0 {
		t.Errorf("dump() = %v, want none left after reset()", got)
	}
}

func TestMessageContextDumpCopiesBufferedMessages(t *testing.T) {
	messageContext := &postgresMessageContext{messages: map[string]string{}}
	messageContext.append("trace", "abc")

	dumped := messageContext.dump()
	dumped["trace"] = "tampered"

	if got := messageContext.dump(); got["trace"] != "abc" {
		t.Errorf("dump() = %v, want a copy the caller cannot write through", got)
	}
}

func TestMessageContextDumpReturnsEmptyMapWhenEmpty(t *testing.T) {
	messageContext := &postgresMessageContext{messages: map[string]string{}}

	got := messageContext.dump()
	if got == nil {
		t.Fatal("dump() = nil, want a non-nil empty map")
	}

	if len(got) != 0 {
		t.Errorf("dump() = %v, want none buffered", got)
	}

	if again := messageContext.dump(); len(again) != 0 {
		t.Errorf("dump() = %v, want none buffered on a repeated call", again)
	}
}

func BenchmarkMessageContextDump(b *testing.B) {
	benchmarks := []struct {
		name     string
		messages map[string]string
	}{
		{"empty", map[string]string{}},
		{"one entry", map[string]string{"trace": "abc"}},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			messageContext := &postgresMessageContext{messages: maps.Clone(bm.messages)}

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				sinkMessages = messageContext.dump()
			}
		})
	}
}

// testStandbyInterval is the standby update interval every test wal runs with,
// short enough that a keepalive tick lands inside a test's patience.
const testStandbyInterval = 20 * time.Millisecond

// sinkMessages keeps the benchmarked dump() result reachable so the compiler
// cannot elide the call.
var sinkMessages map[string]string

// handlerCall is one dispatch the process handler received.
type handlerCall struct {
	timestamp time.Time
	operation domain.OperationType
	dbName    domain.DbName
	schema    domain.TableSchema
	table     domain.TableName
	headers   domain.Headers
	prev      domain.DbRecord
	next      domain.DbRecord
}

// handlerRecorder is a process handler that records every dispatch and answers
// with a fixed error.
type handlerRecorder struct {
	mu    sync.Mutex
	calls []handlerCall
	err   error
}

// newHandlerRecorder returns a recorder whose handler answers with err.
func newHandlerRecorder(err error) *handlerRecorder {
	return &handlerRecorder{err: err}
}

// handle implements database.ProcessMessageHandler.
func (r *handlerRecorder) handle(
	_ context.Context,
	timestamp time.Time,
	operation domain.OperationType,
	dbName domain.DbName,
	schema domain.TableSchema,
	table domain.TableName,
	headers domain.Headers,
	prev domain.DbRecord,
	next domain.DbRecord,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, handlerCall{
		timestamp: timestamp,
		operation: operation,
		dbName:    dbName,
		schema:    schema,
		table:     table,
		headers:   headers,
		prev:      prev,
		next:      next,
	})

	return r.err
}

// recorded returns the dispatches the handler received, in order.
func (r *handlerRecorder) recorded() []handlerCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]handlerCall(nil), r.calls...)
}

// usersServer returns a fake server whose schema holds the single table
// public.users.
func usersServer(t testing.TB) *fakeServer {
	t.Helper()

	s := newFakeServer(t)
	s.setResponder(tableLayoutResponder(false, layoutRow("public", "users", tableTypeRegular, nil, nil)))

	return s
}

// twoTableServer returns a fake server whose schema holds public.users and
// public.orders, which is what a truncate covering both needs to resolve.
func twoTableServer(t *testing.T) *fakeServer {
	t.Helper()

	s := newFakeServer(t)
	s.setResponder(tableLayoutResponder(false,
		layoutRow("public", "users", tableTypeRegular, nil, nil),
		layoutRow("public", "orders", tableTypeRegular, nil, nil),
	))

	return s
}

// newTestWal returns a wal holding the relation cache and type map Start builds,
// resolving tables through an introspector backed by s. It has no replication
// session, so only the decoding half of the wal can be exercised on it.
func newTestWal(t *testing.T, s *fakeServer, tables ...config.Table) (*wal, *fakeHealth) {
	t.Helper()

	health := newFakeHealth()

	return &wal{
		dbName:          "db",
		relations:       map[uint32]*pglogrepl.RelationMessageV2{},
		typeMap:         pgtype.NewMap(),
		messageContext:  &postgresMessageContext{messages: map[string]string{}},
		introspector:    newTestIntrospector(t, s, tables, nil),
		standbyInterval: testStandbyInterval,
		health:          health,
		metrics:         newTestMetrics(),
	}, health
}

// newReplicationWal returns a wal holding a replication session on s, taking
// ownership of conn.
func newReplicationWal(t testing.TB, s *fakeServer, conn *connection, tables ...config.Table) (*wal, *fakeHealth) {
	t.Helper()

	health := newFakeHealth()

	w, err := newWal(context.Background(), conn, "hump_pub", "hump_slot", newTestIntrospector(t, s, tables, nil), health, newTestMetrics())
	if err != nil {
		t.Fatalf("newWal() error = %v", err)
	}

	return w, health
}

// registerRelation puts a relation in w's cache the way a relation message on
// the stream would.
func registerRelation(t *testing.T, w *wal, relationId uint32, schemaName, name string, columns ...relationColumn) {
	t.Helper()

	message := relationMessage(relationId, schemaName, name, columns...)
	if err := w.processMessage(context.Background(), message, time.Now(), 0, failingHandler(t)); err != nil {
		t.Fatalf("processMessage(relation) error = %v", err)
	}
}

// idColumn is the int4 primary key every fixture relation carries.
func idColumn() relationColumn {
	return relationColumn{name: "id", oid: pgtype.Int4OID, key: true}
}

// blobColumn is the TOASTable text column the unchanged-TOAST tests exercise.
func blobColumn() relationColumn {
	return relationColumn{name: "blob", oid: pgtype.TextOID}
}

// tupleData builds the tuple submessage decodeRow reads one row from.
func tupleData(columns ...tupleColumn) *pglogrepl.TupleData {
	tuple := &pglogrepl.TupleData{ColumnNum: uint16(len(columns))}

	for _, column := range columns {
		tuple.Columns = append(tuple.Columns, &pglogrepl.TupleDataColumn{
			DataType: column.dataType,
			Length:   uint32(len(column.data)),
			Data:     []byte(column.data),
		})
	}

	return tuple
}

// runStart runs w.Start in the background and returns the channel its error
// arrives on.
func runStart(ctx context.Context, w *wal, handler database.ProcessMessageHandler) <-chan error {
	errs := make(chan error, 1)
	go func() { errs <- w.Start(ctx, handler) }()

	return errs
}

// awaitStart returns the error Start finished with, failing the test if it does
// not return within five seconds.
func awaitStart(t *testing.T, errs <-chan error) error {
	t.Helper()

	select {
	case err := <-errs:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Start to return")

		return nil
	}
}

// unusedAddress returns a loopback address nothing is listening on, so a dial
// to it fails immediately.
func unusedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	return address
}

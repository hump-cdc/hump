package postgres

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
)

func TestNewExposesPublicationAndSlot(t *testing.T) {
	p := New("postgres://u:p@127.0.0.1:5432/db?sslmode=disable", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	if got := p.GetPublicationName(); got != "hump_pub" {
		t.Errorf("GetPublicationName() = %q, want %q", got, "hump_pub")
	}

	if got := p.GetSlotName(); got != "hump_slot" {
		t.Errorf("GetSlotName() = %q, want %q", got, "hump_slot")
	}
}

func TestPostgresReadsConnectionStringParts(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantName string
		wantHost string
		wantPort uint16
	}{
		{"parseable", "postgres://u:p@127.0.0.1:6432/orders?sslmode=disable", "orders", "127.0.0.1", 6432},
		{"unparseable", "mysql://u@127.0.0.1/orders", "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.url, config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

			if got := p.GetDbName(); got != tt.wantName {
				t.Errorf("GetDbName() = %q, want %q", got, tt.wantName)
			}

			if got := p.GetDbHost(); got != tt.wantHost {
				t.Errorf("GetDbHost() = %q, want %q", got, tt.wantHost)
			}

			if got := p.GetDbPort(); got != tt.wantPort {
				t.Errorf("GetDbPort() = %d, want %d", got, tt.wantPort)
			}
		})
	}
}

func TestValidateCompatibility(t *testing.T) {
	tests := []struct {
		name      string
		result    queryResult
		wantErr   string
		wantSetup bool
	}{
		{name: "supported", result: supportedDatabaseConfig()},
		{name: "unreadable settings", result: serverError("42501", "permission denied"), wantErr: "permission denied"},
		{name: "no result", result: queryResult{empty: true}, wantErr: "empty database config result", wantSetup: true},
		{name: "no rows", result: queryResult{columns: 4}, wantErr: "empty database config result", wantSetup: true},
		{name: "unreadable version", result: databaseConfig("seventeen", "logical", "10", "10"), wantErr: "parse server_version_num", wantSetup: true},
		{name: "old version", result: databaseConfig("140012", "logical", "10", "10"), wantErr: "unsupported version", wantSetup: true},
		{name: "wrong wal level", result: databaseConfig("170000", "replica", "10", "10"), wantErr: "unsupported wal_level", wantSetup: true},
		{name: "unreadable slot limit", result: databaseConfig("170000", "logical", "many", "10"), wantErr: "parse max_replication_slots", wantSetup: true},
		{name: "no slots", result: databaseConfig("170000", "logical", "0", "10"), wantErr: "max_replication_slots too low", wantSetup: true},
		{name: "unreadable sender limit", result: databaseConfig("170000", "logical", "10", "many"), wantErr: "parse max_wal_senders", wantSetup: true},
		{name: "no senders", result: databaseConfig("170000", "logical", "10", "0"), wantErr: "max_wal_senders too low", wantSetup: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(func(string) queryResult { return tt.result })

			p, _, _ := newTestPostgres(t, server)

			err := p.validateCompatibility(context.Background(), newTestConnection(t, server))

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateCompatibility() error = %v, want nil", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateCompatibility() error = %v, want it to mention %q", err, tt.wantErr)
			}

			if got := errors.Is(err, database.ErrDbSetup); got != tt.wantSetup {
				t.Errorf("errors.Is(err, ErrDbSetup) = %t, want %t", got, tt.wantSetup)
			}
		})
	}
}

func TestConfigureBootstrapsPublicationAndSlot(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(configureResponder(layoutRow("public", "users", tableTypeRegular, nil, nil)))

	p, _, _ := newTestPostgres(t, server, configuredTable("public", "users"))

	introspector, err := p.configure(context.Background(), newTestConnection(t, server))
	if err != nil {
		t.Fatalf("configure() error = %v", err)
	}

	if introspector == nil {
		t.Fatal("configure() returned a nil introspector")
	}

	statements := strings.Join(server.recorded(), "\n")
	if !strings.Contains(statements, `CREATE PUBLICATION "hump_pub"`) {
		t.Errorf("configure() did not create the publication, statements:\n%s", statements)
	}

	if !strings.Contains(statements, "pg_create_logical_replication_slot") {
		t.Errorf("configure() did not create the replication slot, statements:\n%s", statements)
	}

	if !strings.Contains(statements, "pg_publication_tables") {
		t.Errorf("configure() did not check replica identity of published tables, statements:\n%s", statements)
	}
}

func TestConfigureReportsFailingStage(t *testing.T) {
	tests := []struct {
		name    string
		marker  string
		wantErr string
	}{
		{"compatibility", "server_version_num", "validate database compatibility"},
		{"introspection", "unnest(", "invalidate introspection"},
		{"publication", "CREATE PUBLICATION", "ensure publication"},
		{"replication slot", "pg_create_logical_replication_slot", "ensure replication slot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(failingOn(
				configureResponder(layoutRow("public", "users", tableTypeRegular, nil, nil)),
				tt.marker,
				serverError("42501", "permission denied"),
			))

			p, _, _ := newTestPostgres(t, server, configuredTable("public", "users"))

			_, err := p.configure(context.Background(), newTestConnection(t, server))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("configure() error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestCleanupDropsPublicationAndSlot(t *testing.T) {
	server := newFakeServer(t)

	p, _, _ := newTestPostgres(t, server)

	if err := p.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	statements := strings.Join(server.recorded(), "\n")
	if !strings.Contains(statements, `DROP PUBLICATION IF EXISTS "hump_pub"`) {
		t.Errorf("Cleanup() did not drop the publication, statements:\n%s", statements)
	}

	if !strings.Contains(statements, "pg_drop_replication_slot") {
		t.Errorf("Cleanup() did not drop the replication slot, statements:\n%s", statements)
	}
}

func TestCleanupRejectsUnusableConnectionString(t *testing.T) {
	p := New("mysql://u@127.0.0.1/db", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	if err := p.Cleanup(context.Background()); err == nil {
		t.Fatal("Cleanup() error = nil, want an error")
	}
}

func TestCleanupReportsFailingStage(t *testing.T) {
	tests := []struct {
		name    string
		marker  string
		wantErr string
	}{
		{"publication", "DROP PUBLICATION", "remove publication"},
		{"replication slot", "pg_drop_replication_slot", "remove replication slot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.setResponder(failingOn(
				func(string) queryResult { return queryResult{} },
				tt.marker,
				serverError("42501", "permission denied"),
			))

			p, _, _ := newTestPostgres(t, server)

			err := p.Cleanup(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Cleanup() error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestListenRejectsUnusableConnectionString(t *testing.T) {
	p := New("mysql://u@127.0.0.1/db", config.Configuration{}, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(nil))

	err := p.Listen(context.Background(), failingHandler(t))
	if err == nil || !strings.Contains(err.Error(), "unable to connect to database") {
		t.Fatalf("Listen() error = %v, want it to mention an unusable connection string", err)
	}
}

func TestListenFailsWhenDatabaseCannotBeConfigured(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(func(string) queryResult { return serverError("42501", "permission denied") })

	p, healthPostgres, _ := newTestPostgres(t, server)

	err := p.Listen(context.Background(), failingHandler(t))
	if err == nil || !strings.Contains(err.Error(), "unable to configure to database") {
		t.Fatalf("Listen() error = %v, want it to mention a failed configuration", err)
	}

	if got := healthPostgres.healthyCount(); got != 0 {
		t.Errorf("MarkHealthy() called %d times, want 0", got)
	}
}

func TestListenFailsWhenReplicationSessionCannotOpen(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(configureResponder(layoutRow("public", "users", tableTypeRegular, nil, nil)))
	server.refuseReplication()

	p, healthPostgres, _ := newTestPostgres(t, server, configuredTable("public", "users"))

	err := p.Listen(context.Background(), failingHandler(t))
	if err == nil || !strings.Contains(err.Error(), "failed to initialize WAL processing") {
		t.Fatalf("Listen() error = %v, want it to mention a failed WAL initialization", err)
	}

	if got := healthPostgres.healthyCount(); got != 1 {
		t.Errorf("MarkHealthy() called %d times, want 1", got)
	}
}

func TestListenStreamsChangesUntilContextIsCancelled(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(configureResponder(layoutRow("public", "users", tableTypeRegular, nil, nil)))

	p, healthPostgres, healthWal := newTestPostgres(t, server, configuredTable("public", "users"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type change struct {
		operation domain.OperationType
		schema    domain.TableSchema
		table     domain.TableName
		next      domain.DbRecord
	}
	changes := make(chan change, 1)

	errs := make(chan error, 1)
	go func() {
		errs <- p.Listen(ctx, func(_ context.Context, _ time.Time, operation domain.OperationType, _ domain.DbName, schema domain.TableSchema, table domain.TableName, _ domain.Headers, _ database.MessageMetadata, _ domain.DbRecord, next domain.DbRecord) error {
			changes <- change{operation: operation, schema: schema, table: table, next: next}
			cancel()

			return nil
		})
	}()

	server.stream(xLogData(0x10, 0x100, relationMessage(1, "public", "users", relationColumn{name: "id", oid: pgtype.Int4OID, key: true})))
	server.stream(xLogData(0x20, 0x100, insertMessage(1, textColumn("7"))))

	select {
	case got := <-changes:
		if got.operation != domain.OperationTypeInsert || got.schema != "public" || got.table != "users" {
			t.Errorf("handler got %v %s.%s, want insert public.users", got.operation, got.schema, got.table)
		}

		if got.next["id"] != int32(7) {
			t.Errorf(`handler got next["id"] = %#v, want int32(7)`, got.next["id"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the change to reach the handler")
	}

	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Listen() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Listen to return")
	}

	if got := healthPostgres.healthyCount(); got != 1 {
		t.Errorf("healthPostgres.MarkHealthy() called %d times, want 1", got)
	}

	if got := healthWal.healthyCount(); got != 1 {
		t.Errorf("healthWal.MarkHealthy() called %d times, want 1", got)
	}

	if failures := healthWal.failures(); len(failures) == 0 || !errors.Is(failures[len(failures)-1], database.ErrWalClosed) {
		t.Errorf("healthWal failures = %v, want the last one to be ErrWalClosed", failures)
	}
}

func TestListenReusesMetricsAcrossCalls(t *testing.T) {
	server := newFakeServer(t)
	server.setResponder(configureResponder(layoutRow("public", "users", tableTypeRegular, nil, nil)))

	p := newRegisteringTestPostgres(t, server, configuredTable("public", "users"))

	// The collectors are registered on a real registry, so a second Listen that
	// rebuilt them instead of reusing the ones from New would panic here.
	listenOnce(t, p, server, 1)
	listenOnce(t, p, server, 2)
}

// queryResult is the fake server's scripted answer to one statement.
type queryResult struct {
	stringSetting string
	// columns is how many text fields the RowDescription announces.
	columns int
	// rows holds the text-format value of every column; a nil value is NULL.
	rows [][][]byte
	// err, when set, replaces the result with an ErrorResponse.
	err *pgproto3.ErrorResponse
	// empty replies with EmptyQueryResponse, so the caller sees no result.
	empty bool
	// copyOut, when set, answers with a COPY TO STDOUT stream of these CopyData
	// payloads instead of rows.
	copyOut [][]byte
	// statements, when set, answers a multi-statement query with one result
	// per entry, in order, instead of this one.
	statements []queryResult
}

// fakeServer is an in-process PostgreSQL server that answers every statement
// from a scripted responder, streams logical replication from a queue of
// frames, and records what it was asked.
type fakeServer struct {
	listener net.Listener

	mu             sync.Mutex
	respond        func(sql string) queryResult
	statements     []string
	params         [][][]byte
	standby        []pglogrepl.StandbyStatusUpdate
	stringSetting  *string
	startupParams  []map[string]string
	closedSessions int
	// refused drops every session that asks for replication=database, so the
	// client's dial never completes.
	refused bool
	// startError, when set, refuses START_REPLICATION with it.
	startError *pgproto3.ErrorResponse

	// frames carries the messages streamed to the client once replication has
	// started.
	frames chan pgproto3.BackendMessage
}

// newFakeServer starts a server on loopback that answers every statement with
// a bare success until setResponder replaces the script.
func newFakeServer(t testing.TB) *fakeServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	s := &fakeServer{
		listener: listener,
		respond:  func(string) queryResult { return queryResult{} },
		frames:   make(chan pgproto3.BackendMessage, 16),
	}
	go s.accept()

	return s
}

// setResponder installs the script that answers every later statement.
func (s *fakeServer) setResponder(respond func(sql string) queryResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.respond = respond
}

// refuseReplication drops every later session that asks for replication, which
// is what a server without a free WAL sender looks like to the client.
func (s *fakeServer) refuseReplication() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refused = true
}

// failReplicationStart answers START_REPLICATION with the given SQLSTATE and
// message instead of entering copy-both mode.
func (s *fakeServer) failReplicationStart(code, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startError = &pgproto3.ErrorResponse{Severity: "ERROR", Code: code, Message: message}
}

// stream queues one frame for delivery to the replication client.
func (s *fakeServer) stream(message pgproto3.BackendMessage) {
	s.frames <- message
}

// recorded returns the SQL the server has been asked to run, in order.
func (s *fakeServer) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.statements...)
}

// boundParams returns the parameters of every extended-protocol Bind, in order.
func (s *fakeServer) boundParams() [][][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([][][]byte(nil), s.params...)
}

// standbyUpdates returns the standby status updates the replication client has
// sent, in order.
func (s *fakeServer) standbyUpdates() []pglogrepl.StandbyStatusUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]pglogrepl.StandbyStatusUpdate(nil), s.standby...)
}

// waitForStandbyUpdates blocks until at least n standby status updates have
// arrived, failing the test if they do not within five seconds.
func (s *fakeServer) waitForStandbyUpdates(t testing.TB, n int) []pglogrepl.StandbyStatusUpdate {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if updates := s.standbyUpdates(); len(updates) >= n {
			return updates
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d standby status updates", n)

	return nil
}

// waitForStandbyPosition blocks until an update reporting position arrives,
// failing the test after five seconds. Matching on the position rather than a
// count keeps an unrelated keepalive tick from satisfying the caller.
func (s *fakeServer) waitForStandbyPosition(t testing.TB, position pglogrepl.LSN) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, update := range s.standbyUpdates() {
			if update.WALWritePosition == position {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timed out waiting for a standby status update reporting %s", position)
}

// accept serves every session until the listener is closed by the cleanup.
func (s *fakeServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		go s.serve(conn)
	}
}

// serve completes the startup handshake and then answers simple and extended
// protocol messages until the client goes away. START_REPLICATION hands the
// session over to copyBoth and ends it there.
func (s *fakeServer) serve(conn net.Conn) {
	defer func() {
		_ = conn.Close()
		s.mu.Lock()
		s.closedSessions++
		s.mu.Unlock()
	}()

	backend := pgproto3.NewBackend(conn, conn)
	if err := s.startup(backend); err != nil {
		return
	}

	// The extended protocol splits one statement across Parse..Sync, so the
	// answer scripted at Parse has to survive until Execute asks for the rows.
	var pending queryResult

	for {
		message, err := backend.Receive()
		if err != nil {
			return
		}

		switch m := message.(type) {
		case *pgproto3.Terminate:
			return
		case *pgproto3.Query:
			if strings.HasPrefix(m.String, "START_REPLICATION") {
				s.record(m.String)

				if startError := s.replicationStartError(); startError != nil {
					backend.Send(startError)
					backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})

					break
				}

				backend.Send(&pgproto3.CopyBothResponse{})
				if err := backend.Flush(); err != nil {
					return
				}

				s.copyBoth(backend)

				return
			}

			result := s.record(m.String)
			parts := result.statements
			if parts == nil {
				parts = []queryResult{result}
			}
			for _, part := range parts {
				if part.columns > 0 {
					backend.Send(rowDescription(part.columns))
				}
				sendRows(backend, part)
				if part.err != nil {
					break
				}
			}
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		case *pgproto3.Parse:
			pending = s.record(m.Query)
			backend.Send(&pgproto3.ParseComplete{})
		case *pgproto3.Bind:
			s.recordParams(m.Parameters)
			backend.Send(&pgproto3.BindComplete{})
		case *pgproto3.Describe:
			backend.Send(rowDescription(max(pending.columns, 1)))
		case *pgproto3.Execute:
			sendRows(backend, pending)
		case *pgproto3.Sync:
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		}

		if err := backend.Flush(); err != nil {
			return
		}
	}
}

// copyBoth streams the queued frames to the client while recording the standby
// status updates it sends back, returning once the stream is closed or the
// client goes away.
func (s *fakeServer) copyBoth(backend *pgproto3.Backend) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			message, err := backend.Receive()
			if err != nil {
				return
			}

			if data, ok := message.(*pgproto3.CopyData); ok {
				s.recordStandby(data.Data)
			}
		}
	}()

	for {
		select {
		case frame := <-s.frames:
			backend.Send(frame)
			if err := backend.Flush(); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// record stores sql and returns the answer the script has for it.
func (s *fakeServer) record(sql string) queryResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statements = append(s.statements, sql)

	return s.respond(sql)
}

// recordParams stores a copy of the parameters of one Bind, whose values alias
// the backend's read buffer and are overwritten by the next message.
func (s *fakeServer) recordParams(params [][]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := make([][]byte, len(params))
	for i, value := range params {
		copied[i] = bytes.Clone(value)
	}

	s.params = append(s.params, copied)
}

// recordStandby stores the standby status update a client copy frame carries,
// ignoring every other copy message.
func (s *fakeServer) recordStandby(data []byte) {
	if len(data) != 34 || data[0] != pglogrepl.StandbyStatusUpdateByteID {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.standby = append(s.standby, pglogrepl.StandbyStatusUpdate{
		WALWritePosition: pglogrepl.LSN(binary.BigEndian.Uint64(data[1:])),
		WALFlushPosition: pglogrepl.LSN(binary.BigEndian.Uint64(data[9:])),
		WALApplyPosition: pglogrepl.LSN(binary.BigEndian.Uint64(data[17:])),
		ReplyRequested:   data[33] != 0,
	})
}

// replicationRefused reports whether replication sessions are being dropped.
func (s *fakeServer) replicationRefused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.refused
}

// replicationStartError returns the error START_REPLICATION is scripted to
// answer with, or nil when the stream should start.
func (s *fakeServer) replicationStartError() *pgproto3.ErrorResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.startError
}

// startup answers the startup message, leaving the backend ready for queries.
// A session asking for replication while replication is refused is dropped
// before the handshake completes, so the client's dial fails, and so is a
// message that is not a startup (a cancel request, say).
func (s *fakeServer) startup(backend *pgproto3.Backend) error {
	message, err := backend.ReceiveStartupMessage()
	if err != nil {
		return err
	}

	startup, ok := message.(*pgproto3.StartupMessage)
	if !ok {
		return fmt.Errorf("unexpected startup message %T", message)
	}

	if startup.Parameters["replication"] == "database" && s.replicationRefused() {
		return errors.New("replication refused")
	}
	s.mu.Lock()
	s.startupParams = append(s.startupParams, startup.Parameters)
	setting := "on"
	if s.stringSetting != nil {
		setting = *s.stringSetting
	}
	s.mu.Unlock()

	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "17.0"})
	backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: setting})
	backend.Send(&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: []byte{0, 0, 0, 2}})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})

	return backend.Flush()
}

// sendRows writes the body of a result: an error, an empty query response, or
// the data rows followed by their command tag.
func sendRows(backend *pgproto3.Backend, result queryResult) {
	if result.stringSetting != "" {
		backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: result.stringSetting})
	}
	switch {
	case result.err != nil:
		backend.Send(result.err)
	case result.empty:
		backend.Send(&pgproto3.EmptyQueryResponse{})
	case result.copyOut != nil:
		backend.Send(&pgproto3.CopyOutResponse{})
		for _, data := range result.copyOut {
			backend.Send(&pgproto3.CopyData{Data: data})
		}
		backend.Send(&pgproto3.CopyDone{})
		backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("COPY " + strconv.Itoa(len(result.copyOut)))})
	default:
		for _, row := range result.rows {
			backend.Send(&pgproto3.DataRow{Values: row})
		}
		backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT " + strconv.Itoa(len(result.rows)))})
	}
}

// rowDescription announces columns text fields named col0, col1 and so on.
func rowDescription(columns int) *pgproto3.RowDescription {
	fields := make([]pgproto3.FieldDescription, columns)
	for i := range fields {
		fields[i] = pgproto3.FieldDescription{
			Name:         fmt.Appendf(nil, "col%d", i),
			DataTypeOID:  25,
			DataTypeSize: -1,
		}
	}

	return &pgproto3.RowDescription{Fields: fields}
}

// tableLayoutResponder answers the timescaledb probe with timescale and the
// table layout query with rows, leaving every other statement a bare success.
func tableLayoutResponder(timescale bool, rows ...[][]byte) func(string) queryResult {
	flag := "f"
	if timescale {
		flag = "t"
	}

	return func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "pg_extension"):
			return singleValue(flag)
		case strings.Contains(sql, "inheritance parent"):
			return inheritanceParent()
		case strings.Contains(sql, "pg_publication_tables"):
			return publishedTables()
		case strings.Contains(sql, "relreplident"):
			return staleReplicaIdentity()
		case strings.Contains(sql, "unnest("):
			return tableLayout(rows...)
		default:
			return queryResult{}
		}
	}
}

// stagedLayoutResponder answers the n-th layout query with the n-th stage,
// repeating the last one once the stages run out. It scripts a schema that
// changes under a running introspector.
func stagedLayoutResponder(stages ...[][][]byte) func(string) queryResult {
	var mu sync.Mutex
	var seen int

	return func(sql string) queryResult {
		switch {
		case strings.Contains(sql, "pg_extension"):
			return singleValue("f")
		case strings.Contains(sql, "inheritance parent"):
			return inheritanceParent()
		case strings.Contains(sql, "pg_publication_tables"):
			return publishedTables()
		case strings.Contains(sql, "relreplident"):
			return staleReplicaIdentity()
		case strings.Contains(sql, "unnest("):
			mu.Lock()
			defer mu.Unlock()

			stage := stages[min(seen, len(stages)-1)]
			seen++

			return tableLayout(stage...)
		default:
			return queryResult{}
		}
	}
}

// configureResponder answers every statement a successful configure runs: the
// compatibility probe, the timescaledb probe and the table layout, leaving the
// bootstrapping statements a bare success.
func configureResponder(rows ...[][]byte) func(string) queryResult {
	layout := tableLayoutResponder(false, rows...)

	return func(sql string) queryResult {
		if strings.Contains(sql, "server_version_num") {
			return supportedDatabaseConfig()
		}

		return layout(sql)
	}
}

// failingOn wraps respond so that every statement containing marker is answered
// with result instead.
func failingOn(respond func(string) queryResult, marker string, result queryResult) func(string) queryResult {
	return func(sql string) queryResult {
		if strings.Contains(sql, marker) {
			return result
		}

		return respond(sql)
	}
}

// withParent wraps respond so that the inheritance parent lookup answers with
// parent instead of the empty result the layout responders return.
func withParent(respond func(string) queryResult, parent domain.TableIdentifier) func(string) queryResult {
	return func(sql string) queryResult {
		if strings.Contains(sql, "inheritance parent") {
			return inheritanceParent(parent)
		}

		return respond(sql)
	}
}

// inheritanceParent scripts the result of the parent lookup, whose single row is
// the parent of the table asked about. No table means no parent.
func inheritanceParent(tables ...domain.TableIdentifier) queryResult {
	rows := make([][][]byte, 0, len(tables))
	for _, t := range tables {
		rows = append(rows, [][]byte{[]byte(t.Schema), []byte(t.Name)})
	}

	return queryResult{columns: 2, rows: rows}
}

// publishedTables scripts the result of the replica identity preflight, whose
// rows are the published tables that support neither UPDATE nor DELETE.
func publishedTables(tables ...domain.TableIdentifier) queryResult {
	rows := make([][][]byte, 0, len(tables))
	for _, t := range tables {
		rows = append(rows, [][]byte{[]byte(t.Schema), []byte(t.Name)})
	}

	return queryResult{columns: 2, rows: rows}
}

// singleValue scripts a one-column, one-row result holding value.
func singleValue(value string) queryResult {
	return queryResult{columns: 1, rows: [][][]byte{{[]byte(value)}}}
}

// databaseConfig scripts the result of config.sql, whose columns are
// server_version_num, wal_level, max_replication_slots and max_wal_senders.
func databaseConfig(versionNum, walLevel, replicationSlots, walSenders string) queryResult {
	return queryResult{
		columns: 4,
		rows:    [][][]byte{{[]byte(versionNum), []byte(walLevel), []byte(replicationSlots), []byte(walSenders)}},
	}
}

// supportedDatabaseConfig scripts a config.sql result that meets every
// requirement for logical replication.
func supportedDatabaseConfig() queryResult {
	return databaseConfig("170000", "logical", "10", "10")
}

// tableLayout scripts the result of the table layout query, taking its column
// count from the first row.
func tableLayout(rows ...[][]byte) queryResult {
	columns := 5
	if len(rows) > 0 {
		columns = len(rows[0])
	}

	return queryResult{columns: columns, rows: rows}
}

// staleReplicaIdentity scripts the result of the replica identity check, whose
// rows are the tables that are not yet FULL.
func staleReplicaIdentity(tables ...domain.TableIdentifier) queryResult {
	rows := make([][][]byte, 0, len(tables))
	for _, t := range tables {
		rows = append(rows, [][]byte{[]byte(t.Schema), []byte(t.Name)})
	}

	return queryResult{columns: 2, rows: rows}
}

// layoutRow builds one row of the table layout query. A nil partitionSchema or
// partitionTable is sent as SQL NULL.
func layoutRow(
	schemaName, name string,
	tableType tableType,
	partitionSchema, partitionTable []byte,
) [][]byte {
	return [][]byte{[]byte(schemaName), []byte(name), []byte(tableTypeLiteral(tableType)), partitionSchema, partitionTable}
}

// tableTypeLiteral renders a tableType as the literal the table layout query
// emits for it.
func tableTypeLiteral(t tableType) string {
	switch t {
	case tableTypeHypertable:
		return "hypertable"
	case tableTypePartitioned:
		return "partitioned"
	default:
		return "regular"
	}
}

// serverError scripts an ErrorResponse carrying the given SQLSTATE and message.
func serverError(code, message string) queryResult {
	return queryResult{err: &pgproto3.ErrorResponse{Severity: "ERROR", Code: code, Message: message}}
}

// replicaIdentityRecorder returns a stub ensureReplicaIdentity that records the
// tables of every call and always succeeds.
func replicaIdentityRecorder() (func(context.Context, *connection, []domain.TableIdentifier) error, *[][]domain.TableIdentifier) {
	var calls [][]domain.TableIdentifier

	return func(_ context.Context, _ *connection, tables []domain.TableIdentifier) error {
		calls = append(calls, tables)

		return nil
	}, &calls
}

// fakeServerUrl returns the connection string that reaches s.
func fakeServerUrl(s *fakeServer) string {
	return fmt.Sprintf("postgres://u:p@%s/db?sslmode=disable", s.listener.Addr())
}

// newTestConnection returns a connection talking to s over a real socket, with
// retries disabled so a scripted error surfaces on the first attempt.
func newTestConnection(t testing.TB, s *fakeServer, opts ...option) *connection {
	t.Helper()

	conn, err := newConnection(fakeServerUrl(s), append([]option{withMaxAttempts(1)}, opts...)...)
	if err != nil {
		t.Fatalf("newConnection() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

// newTestPostgres returns an adapter replicating tables from s, along with the
// health reporters it writes to.
func newTestPostgres(t *testing.T, s *fakeServer, tables ...config.Table) (*Postgres, *fakeHealth, *fakeHealth) {
	t.Helper()

	healthPostgres, healthWal := newFakeHealth(), newFakeHealth()
	cfg := config.Configuration{Tables: tables}

	return New(fakeServerUrl(s), cfg, "hump_pub", "hump_slot", healthPostgres, healthWal, promauto.With(nil)), healthPostgres, healthWal
}

// newRegisteringTestPostgres returns an adapter whose metrics land on a private
// registry, so a duplicate registration panics rather than being swallowed by
// the nil registerer the other helpers use.
func newRegisteringTestPostgres(t *testing.T, s *fakeServer, tables ...config.Table) *Postgres {
	t.Helper()

	cfg := config.Configuration{Tables: tables}

	return New(fakeServerUrl(s), cfg, "hump_pub", "hump_slot", newFakeHealth(), newFakeHealth(), promauto.With(prometheus.NewRegistry()))
}

// listenOnce runs one Listen cycle against s, cancelling it once the WAL
// session has opened, and fails the test if it does not open within five seconds.
func listenOnce(t *testing.T, p *Postgres, s *fakeServer, want int) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error, 1)
	go func() {
		errs <- p.Listen(ctx, failingHandler(t))
	}()

	deadline := time.Now().Add(5 * time.Second)
	for replicationStarts(s) < want {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for START_REPLICATION #%d", want)
		}

		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Listen() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Listen to return")
	}
}

// replicationStarts counts the START_REPLICATION statements s has been asked to run.
func replicationStarts(s *fakeServer) int {
	count := 0
	for _, statement := range s.recorded() {
		if strings.HasPrefix(statement, "START_REPLICATION") {
			count++
		}
	}

	return count
}

// newTestBootstrapper returns a bootstrapper talking to s over a real socket, with
// retries disabled so a scripted error surfaces on the first attempt.
func newTestBootstrapper(t *testing.T, s *fakeServer) *bootstrapper {
	t.Helper()

	return newBootstrapper(newTestConnection(t, s))
}

// newTestTableMetadataLoader returns a loader talking to s over a real socket, with
// retries disabled so a scripted error surfaces on the first attempt.
func newTestTableMetadataLoader(t *testing.T, s *fakeServer) *tableMetadataLoader {
	t.Helper()

	return newTableMetadataLoader(newTestConnection(t, s))
}

// newTestIntrospector returns an introspector talking to s over a real socket, with
// retries disabled so a scripted error surfaces on the first attempt. A nil
// ensure defaults to a stub that always succeeds.
func newTestIntrospector(
	t testing.TB,
	s *fakeServer,
	tables []config.Table,
	ensure func(context.Context, *connection, []domain.TableIdentifier) error,
) *introspector {
	t.Helper()

	if ensure == nil {
		ensure = func(context.Context, *connection, []domain.TableIdentifier) error { return nil }
	}

	registry := newIntrospectorRegistry(config.Configuration{Tables: tables})

	return newIntrospector(newTestConnection(t, s), registry, ensure)
}

// configuredTable builds a config.Table that the introspector registry counts as
// configured, with replica identity managed by hump.
func configuredTable(schemaName domain.TableSchema, name domain.TableName) config.Table {
	return config.Table{
		Schema:  schemaName,
		Name:    name,
		Replica: domain.ReplicaIdentityFull,
	}
}

// unmanagedTable builds a config.Table whose replica identity hump leaves alone.
func unmanagedTable(schemaName domain.TableSchema, name domain.TableName) config.Table {
	return config.Table{
		Schema:  schemaName,
		Name:    name,
		Replica: domain.ReplicaIdentityUnmanaged,
	}
}

// fakeHealth records the health transitions the adapter reports.
type fakeHealth struct {
	mu        sync.Mutex
	healthy   int
	unhealthy []error
}

// newFakeHealth returns a health reporter that has seen no transitions.
func newFakeHealth() *fakeHealth {
	return &fakeHealth{}
}

// MarkHealthy records one healthy transition.
func (h *fakeHealth) MarkHealthy() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.healthy++
}

// MarkUnhealthy records the error behind one unhealthy transition.
func (h *fakeHealth) MarkUnhealthy(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.unhealthy = append(h.unhealthy, err)
}

// healthyCount returns how often MarkHealthy has been called.
func (h *fakeHealth) healthyCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.healthy
}

// failures returns the errors passed to MarkUnhealthy, in order.
func (h *fakeHealth) failures() []error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]error(nil), h.unhealthy...)
}

// waitForHealthy blocks until MarkHealthy has been called, failing the test if
// it is not within five seconds.
func (h *fakeHealth) waitForHealthy(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if h.healthyCount() > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatal("timed out waiting for the health reporter to be marked healthy")
}

// failingHandler returns a process handler that fails the test if the stream
// ever reaches it.
func failingHandler(t *testing.T) database.ProcessMessageHandler {
	t.Helper()

	return func(context.Context, time.Time, domain.OperationType, domain.DbName, domain.TableSchema, domain.TableName, domain.Headers, database.MessageMetadata, domain.DbRecord, domain.DbRecord) error {
		t.Error("process handler was called, want the stream to stop before it")

		return nil
	}
}

// unencodableCodec is a text[] codec that never builds an encode plan, so every
// parameter routed through it fails to encode.
type unencodableCodec struct{ pgtype.TextCodec }

// PlanEncode reports that no encoding is possible.
func (unencodableCodec) PlanEncode(*pgtype.Map, uint32, int16, any) pgtype.EncodePlan {
	return nil
}

// newUnencodableTypeMap returns a type map whose text[] parameters cannot be encoded.
func newUnencodableTypeMap() *pgtype.Map {
	m := pgtype.NewMap()
	m.RegisterType(&pgtype.Type{Name: "_text", OID: pgtype.TextArrayOID, Codec: unencodableCodec{}})

	return m
}

// fakeDialer replaces the network dial with an in-memory pipe served by a
// backend that completes the startup handshake, and records how many sessions
// were opened and how many dials overlapped.
type fakeDialer struct {
	t    *testing.T
	gate chan struct{}
	// serve backs the pipe for the dial-th session; nil serves serveStartup.
	serve func(conn net.Conn, dial int)

	mu          sync.Mutex
	dials       int
	inFlight    int
	maxInFlight int
}

// newFakeDialer returns a dialer whose sessions connect immediately. A gated
// dialer instead parks every dial until releaseGate is called.
func newFakeDialer(t *testing.T, gated bool) *fakeDialer {
	d := &fakeDialer{t: t}
	if gated {
		d.gate = make(chan struct{})
	}

	return d
}

// dial implements pgconn.DialFunc.
func (d *fakeDialer) dial(ctx context.Context, _, _ string) (net.Conn, error) {
	dial := d.enter()
	defer d.leave()

	if d.gate != nil {
		select {
		case <-d.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	serve := d.serve
	if serve == nil {
		serve = func(conn net.Conn, _ int) { serveStartup(conn) }
	}

	client, server := net.Pipe()
	go serve(server, dial)

	return client, nil
}

// enter records the start of a dial, updates the overlap high-water mark and
// returns the 1-based index of the dial it just started.
func (d *fakeDialer) enter() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.dials++
	d.inFlight++
	if d.inFlight > d.maxInFlight {
		d.maxInFlight = d.inFlight
	}

	return d.dials
}

// leave records the end of a dial.
func (d *fakeDialer) leave() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.inFlight--
}

// counts returns the number of dials started and the highest overlap seen.
func (d *fakeDialer) counts() (int, int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.dials, d.maxInFlight
}

// releaseGate unblocks every parked and future dial.
func (d *fakeDialer) releaseGate() {
	close(d.gate)
}

// waitForDial blocks until at least n dials have started, failing the test if
// they do not within a second.
func (d *fakeDialer) waitForDial(n int) {
	d.t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if started, _ := d.counts(); started >= n {
			return
		}
		time.Sleep(time.Millisecond)
	}

	d.t.Fatalf("timed out waiting for %d dials", n)
}

// pipeStartup answers the startup message on a net.Pipe backend, leaving it
// ready for queries.
func pipeStartup(backend *pgproto3.Backend) error {
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return err
	}

	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "16.0"})
	backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
	backend.Send(&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: []byte{0, 0, 0, 2}})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})

	return backend.Flush()
}

// serveStartup answers the startup message and then drains the pipe. The drain
// is required: net.Pipe is unbuffered, so a backend that stops reading would
// deadlock the Terminate message pgconn writes on Close.
func serveStartup(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	if err := pipeStartup(pgproto3.NewBackend(conn, conn)); err != nil {
		return
	}

	_, _ = io.Copy(io.Discard, conn)
}

// serveSession answers the startup message and then hands every frontend
// message to reply, closing the pipe once reply or the pipe itself fails.
func serveSession(conn net.Conn, reply func(*pgproto3.Backend, pgproto3.FrontendMessage) error) {
	defer func() { _ = conn.Close() }()

	backend := pgproto3.NewBackend(conn, conn)
	if err := pipeStartup(backend); err != nil {
		return
	}

	for {
		message, err := backend.Receive()
		if err != nil {
			return
		}

		if _, terminate := message.(*pgproto3.Terminate); terminate {
			return
		}

		if err := reply(backend, message); err != nil {
			return
		}
	}
}

// singleColumn describes the one int4 column every fake result carries.
func singleColumn() *pgproto3.RowDescription {
	return &pgproto3.RowDescription{
		Fields: []pgproto3.FieldDescription{{
			Name:         []byte("n"),
			DataTypeOID:  23,
			DataTypeSize: 4,
		}},
	}
}

// replyRow answers a simple or extended query with a single row holding "1".
func replyRow(backend *pgproto3.Backend, message pgproto3.FrontendMessage) error {
	switch message.(type) {
	case *pgproto3.Query:
		backend.Send(singleColumn())
		backend.Send(&pgproto3.DataRow{Values: [][]byte{[]byte("1")}})
		backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
		backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})

		return backend.Flush()
	case *pgproto3.Parse:
		backend.Send(&pgproto3.ParseComplete{})
	case *pgproto3.Bind:
		backend.Send(&pgproto3.BindComplete{})
	case *pgproto3.Describe:
		backend.Send(singleColumn())
	case *pgproto3.Execute:
		backend.Send(&pgproto3.DataRow{Values: [][]byte{[]byte("1")}})
		backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
	case *pgproto3.Sync:
		backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})

		return backend.Flush()
	}

	return nil
}

// replyNotice sends a NOTICE ahead of the result replyRow would send.
func replyNotice(backend *pgproto3.Backend, message pgproto3.FrontendMessage) error {
	if _, query := message.(*pgproto3.Query); query {
		backend.Send(&pgproto3.NoticeResponse{Severity: "NOTICE", Code: "00000", Message: "table already exists"})
	}

	return replyRow(backend, message)
}

// replyError answers a simple query with an ErrorResponse, leaving the session
// usable so the caller sees a server error rather than a dropped socket.
func replyError(backend *pgproto3.Backend, message pgproto3.FrontendMessage) error {
	if _, query := message.(*pgproto3.Query); !query {
		return replyRow(backend, message)
	}

	backend.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: "42601", Message: "syntax error"})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})

	return backend.Flush()
}

// heldSession reads the cached session under the lock, without opening one.
func heldSession(conn *connection) *pgconn.PgConn {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	return conn.connection
}

// newFakeConnection returns a connection that dials through d.
func newFakeConnection(t *testing.T, d *fakeDialer, opts ...option) *connection {
	t.Helper()

	conn, err := newConnection("postgres://u:p@127.0.0.1:5432/db?sslmode=disable", opts...)
	if err != nil {
		t.Fatalf("newConnection() error = %v", err)
	}
	conn.connectionConfig.DialFunc = d.dial

	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

// brokenWriteConn is a net.Conn whose writes start failing once it has written
// afterWrites frames, which breaks a session at an exact point in the protocol
// rather than racing a socket close.
type brokenWriteConn struct {
	net.Conn
	afterWrites int32
	writes      atomic.Int32
}

// Write fails once the connection has written afterWrites frames.
func (c *brokenWriteConn) Write(b []byte) (int, error) {
	if c.writes.Add(1) > c.afterWrites {
		return 0, errors.New("write: broken pipe")
	}

	return c.Conn.Write(b)
}

// brokenWriteDialer returns a DialFunc whose sessions stop accepting writes
// after afterWrites of them.
func brokenWriteDialer(afterWrites int32) func(context.Context, string, string) (net.Conn, error) {
	var dialer net.Dialer

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}

		return &brokenWriteConn{Conn: conn, afterWrites: afterWrites}, nil
	}
}

// errBrokenClose is the error a brokenCloseConn reports from Close.
var errBrokenClose = errors.New("close: broken pipe")

// brokenCloseConn is a net.Conn that closes its socket but reports the close as
// failed.
type brokenCloseConn struct {
	net.Conn
}

// Close closes the socket and returns errBrokenClose.
func (c *brokenCloseConn) Close() error {
	_ = c.Conn.Close()

	return errBrokenClose
}

// brokenCloseDialer is a DialFunc whose sessions fail to close.
func brokenCloseDialer(ctx context.Context, network, address string) (net.Conn, error) {
	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return &brokenCloseConn{Conn: conn}, nil
}

// keepalive builds the primary keepalive frame a WAL sender emits to ask the
// standby where it is.
func keepalive(serverWalEnd uint64, replyRequested bool) *pgproto3.CopyData {
	data := []byte{pglogrepl.PrimaryKeepaliveMessageByteID}
	data = binary.BigEndian.AppendUint64(data, serverWalEnd)
	data = binary.BigEndian.AppendUint64(data, 0)

	reply := byte(0)
	if replyRequested {
		reply = 1
	}

	return &pgproto3.CopyData{Data: append(data, reply)}
}

// xLogData wraps a pgoutput message in the XLogData frame that carries it.
func xLogData(walStart, serverWalEnd uint64, walData []byte) *pgproto3.CopyData {
	data := []byte{pglogrepl.XLogDataByteID}
	data = binary.BigEndian.AppendUint64(data, walStart)
	data = binary.BigEndian.AppendUint64(data, serverWalEnd)
	data = binary.BigEndian.AppendUint64(data, 0)

	return &pgproto3.CopyData{Data: append(data, walData...)}
}

// copyFrame wraps raw bytes in a copy frame, for payloads too short to parse.
func copyFrame(data ...byte) *pgproto3.CopyData {
	return &pgproto3.CopyData{Data: data}
}

// relationColumn is one column of a pgoutput relation message.
type relationColumn struct {
	name string
	oid  uint32
	// key marks the column as part of the replica identity, which is what the
	// server sets on every column of a table with REPLICA IDENTITY FULL.
	key bool
}

// tupleColumn is one column of a pgoutput tuple: 'n' for null, 'u' for an
// unchanged TOAST value, and 't' for a text-format value.
type tupleColumn struct {
	dataType byte
	data     string
}

// textColumn builds a text-format tuple column holding value.
func textColumn(value string) tupleColumn {
	return tupleColumn{dataType: 't', data: value}
}

// nullColumn builds a tuple column holding SQL NULL.
func nullColumn() tupleColumn {
	return tupleColumn{dataType: 'n'}
}

// toastColumn builds a tuple column whose TOAST value the change left unchanged.
func toastColumn() tupleColumn {
	return tupleColumn{dataType: 'u'}
}

// relationMessage encodes the pgoutput message describing a relation's columns,
// which the stream sends before any change to it.
func relationMessage(relationId uint32, namespace, name string, columns ...relationColumn) []byte {
	data := []byte{'R'}
	data = binary.BigEndian.AppendUint32(data, relationId)
	data = appendCString(data, namespace)
	data = appendCString(data, name)
	data = append(data, 'f')
	data = binary.BigEndian.AppendUint16(data, uint16(len(columns)))

	for _, column := range columns {
		flags := byte(0)
		if column.key {
			flags = 1
		}

		data = append(data, flags)
		data = appendCString(data, column.name)
		data = binary.BigEndian.AppendUint32(data, column.oid)
		data = binary.BigEndian.AppendUint32(data, 0)
	}

	return data
}

// insertMessage encodes a pgoutput insert carrying tuple as the new row.
func insertMessage(relationId uint32, tuple ...tupleColumn) []byte {
	data := []byte{'I'}
	data = binary.BigEndian.AppendUint32(data, relationId)
	data = append(data, 'N')

	return appendTuple(data, tuple)
}

// updateMessage encodes a pgoutput update. A nil oldTuple omits the old row,
// which is what a table without REPLICA IDENTITY FULL sends.
func updateMessage(relationId uint32, oldTuple, newTuple []tupleColumn) []byte {
	data := []byte{'U'}
	data = binary.BigEndian.AppendUint32(data, relationId)

	if oldTuple != nil {
		data = append(data, 'O')
		data = appendTuple(data, oldTuple)
	}

	data = append(data, 'N')

	return appendTuple(data, newTuple)
}

// keyUpdateMessage encodes a pgoutput update whose old row is a key-only
// tuple, which is what a table without REPLICA IDENTITY FULL sends when the
// update touches a replica identity column.
func keyUpdateMessage(relationId uint32, keyTuple, newTuple []tupleColumn) []byte {
	data := []byte{'U'}
	data = binary.BigEndian.AppendUint32(data, relationId)
	data = append(data, 'K')
	data = appendTuple(data, keyTuple)
	data = append(data, 'N')

	return appendTuple(data, newTuple)
}

// deleteMessage encodes a pgoutput delete carrying tuple as the old row.
func deleteMessage(relationId uint32, tuple ...tupleColumn) []byte {
	data := []byte{'D'}
	data = binary.BigEndian.AppendUint32(data, relationId)
	data = append(data, 'O')

	return appendTuple(data, tuple)
}

// truncateMessage encodes a pgoutput truncate covering the given relations.
func truncateMessage(relationIds ...uint32) []byte {
	data := []byte{'T'}
	data = binary.BigEndian.AppendUint32(data, uint32(len(relationIds)))
	data = append(data, 0)

	for _, relationId := range relationIds {
		data = binary.BigEndian.AppendUint32(data, relationId)
	}

	return data
}

// beginMessage encodes the pgoutput message opening a transaction.
func beginMessage() []byte {
	data := []byte{'B'}
	data = binary.BigEndian.AppendUint64(data, 0)
	data = binary.BigEndian.AppendUint64(data, 0)

	return binary.BigEndian.AppendUint32(data, 1)
}

// commitMessage encodes the pgoutput message closing a transaction whose WAL
// ends at end.
func commitMessage(end uint64) []byte {
	data := []byte{'C', 0}
	data = binary.BigEndian.AppendUint64(data, 0)
	data = binary.BigEndian.AppendUint64(data, end)

	return binary.BigEndian.AppendUint64(data, 0)
}

// logicalDecodingMessage encodes a pg_logical_emit_message() payload.
func logicalDecodingMessage(transactional bool, prefix, content string) []byte {
	flags := byte(0)
	if transactional {
		flags = 1
	}

	data := []byte{'M', flags}
	data = binary.BigEndian.AppendUint64(data, 0)
	data = appendCString(data, prefix)
	data = binary.BigEndian.AppendUint32(data, uint32(len(content)))

	return append(data, content...)
}

// appendTuple appends a pgoutput TupleData submessage holding columns.
func appendTuple(data []byte, columns []tupleColumn) []byte {
	data = binary.BigEndian.AppendUint16(data, uint16(len(columns)))

	for _, column := range columns {
		data = append(data, column.dataType)

		if column.dataType != 't' {
			continue
		}

		data = binary.BigEndian.AppendUint32(data, uint32(len(column.data)))
		data = append(data, column.data...)
	}

	return data
}

// appendCString appends a null-terminated string, the encoding pgoutput uses
// for every name it carries.
func appendCString(data []byte, value string) []byte {
	return append(append(data, value...), 0)
}

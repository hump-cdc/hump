package postgres

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
)

func TestNewConnectionRejectsInvalidConnectionString(t *testing.T) {
	if _, err := newConnection("mysql://user@localhost/db"); err == nil {
		t.Fatal("expected an error, got nil")
	}

	if _, err := newConnection("mysql://user@localhost/db", withReplication(true)); err == nil {
		t.Fatal("expected an error with withReplication, got nil")
	}
}

func TestNewConnectionDoesNotDial(t *testing.T) {
	conn, err := newConnection("postgres://user@127.0.0.1:1/somedb")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if raw := heldSession(conn); raw != nil {
		t.Fatalf("expected no session before first use, got %v", raw)
	}

	if conn.GetDbName() != "somedb" {
		t.Fatalf("expected database name %q, got %q", "somedb", conn.GetDbName())
	}
}

func TestNewConnectionAppliesReplicationParam(t *testing.T) {
	cases := []struct {
		name             string
		connectionString string
		wantHost         string
		wantPort         uint16
		wantDbName       string
	}{
		{
			name:             "url",
			connectionString: "postgres://u:p@db.example.com:5433/app",
			wantHost:         "db.example.com",
			wantPort:         5433,
			wantDbName:       "app",
		},
		{
			name:             "keyword value dsn",
			connectionString: "host=localhost port=5432 dbname=app user=u",
			wantHost:         "localhost",
			wantPort:         5432,
			wantDbName:       "app",
		},
		{
			name:             "keyword value dsn with quoted password",
			connectionString: "host=localhost port=5432 dbname=app user=u password='se cret'",
			wantHost:         "localhost",
			wantPort:         5432,
			wantDbName:       "app",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := newConnection(tt.connectionString, withReplication(true))
			if err != nil {
				t.Fatalf("newConnection(%q) error = %v", tt.connectionString, err)
			}

			if got := conn.connectionConfig.RuntimeParams["replication"]; got != "database" {
				t.Errorf("RuntimeParams[replication] = %q, want %q", got, "database")
			}

			if got := conn.GetDbHost(); got != tt.wantHost {
				t.Errorf("GetDbHost() = %q, want %q", got, tt.wantHost)
			}
			if got := conn.GetDbPort(); got != tt.wantPort {
				t.Errorf("GetDbPort() = %d, want %d", got, tt.wantPort)
			}
			if got := string(conn.GetDbName()); got != tt.wantDbName {
				t.Errorf("GetDbName() = %q, want %q", got, tt.wantDbName)
			}
		})
	}
}

func TestNewConnectionWithoutReplicationLeavesRuntimeParams(t *testing.T) {
	conn, err := newConnection("postgres://u:p@localhost:5432/app")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, present := conn.connectionConfig.RuntimeParams["replication"]; present {
		t.Fatalf("RuntimeParams[replication] = %q, want it absent", got)
	}
}

func TestNewConnectionUsesDefaultRetryPolicy(t *testing.T) {
	conn, err := newConnection("postgres://u@localhost:5432/app")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conn.maxAttempts != 5 {
		t.Errorf("maxAttempts = %d, want %d", conn.maxAttempts, 5)
	}
	if conn.baseDelay != 100*time.Millisecond {
		t.Errorf("baseDelay = %v, want %v", conn.baseDelay, 100*time.Millisecond)
	}
	if conn.maxDelay != 5*time.Second {
		t.Errorf("maxDelay = %v, want %v", conn.maxDelay, 5*time.Second)
	}
}

func TestNewConnectionOverridesRetryPolicy(t *testing.T) {
	conn, err := newConnection(
		"postgres://u@localhost:5432/app",
		withMaxAttempts(2),
		withBaseDelay(3*time.Millisecond),
		withMaxDelay(7*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conn.maxAttempts != 2 {
		t.Errorf("maxAttempts = %d, want %d", conn.maxAttempts, 2)
	}
	if conn.baseDelay != 3*time.Millisecond {
		t.Errorf("baseDelay = %v, want %v", conn.baseDelay, 3*time.Millisecond)
	}
	if conn.maxDelay != 7*time.Millisecond {
		t.Errorf("maxDelay = %v, want %v", conn.maxDelay, 7*time.Millisecond)
	}
}

func TestNewConnectionIgnoresInvalidRetryPolicy(t *testing.T) {
	cases := []struct {
		name   string
		option option
	}{
		{name: "zero attempts", option: withMaxAttempts(0)},
		{name: "negative attempts", option: withMaxAttempts(-1)},
		{name: "zero base delay", option: withBaseDelay(0)},
		{name: "negative base delay", option: withBaseDelay(-time.Second)},
		{name: "zero max delay", option: withMaxDelay(0)},
		{name: "negative max delay", option: withMaxDelay(-time.Second)},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := newConnection("postgres://u@localhost:5432/app", tt.option)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if conn.maxAttempts != 5 {
				t.Errorf("maxAttempts = %d, want the default %d", conn.maxAttempts, 5)
			}
			if conn.baseDelay != 100*time.Millisecond {
				t.Errorf("baseDelay = %v, want the default %v", conn.baseDelay, 100*time.Millisecond)
			}
			if conn.maxDelay != 5*time.Second {
				t.Errorf("maxDelay = %v, want the default %v", conn.maxDelay, 5*time.Second)
			}
		})
	}
}

func TestNewConnectionErrorHasNoPassword(t *testing.T) {
	cases := []struct {
		name             string
		connectionString string
	}{
		{name: "url", connectionString: "postgres://u:s3cr3t@loc alhost:5432/db"},
		{name: "keyword value dsn", connectionString: "host=localhost port=nope password=s3cr3t dbname=db"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newConnection(tt.connectionString)
			if err == nil {
				t.Fatalf("newConnection(%q) expected an error, got nil", tt.connectionString)
			}

			if strings.Contains(err.Error(), "s3cr3t") {
				t.Errorf("error leaks the password: %v", err)
			}
		})
	}
}

func TestExecReturnsContextErrorWithoutDialing(t *testing.T) {
	conn, err := newConnection(
		"postgres://u@127.0.0.1:1/db",
		withMaxAttempts(3),
		withBaseDelay(time.Second),
		withMaxDelay(time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if _, err := conn.Exec(ctx, "select 1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Exec() error = %v, want it to wrap context.Canceled", err)
	}

	// A cancelled context must short-circuit the whole retry loop, not walk it.
	if elapsed := time.Since(start); elapsed >= conn.baseDelay {
		t.Errorf("Exec() took %v on a cancelled context, want it back before one backoff (%v)", elapsed, conn.baseDelay)
	}
}

func TestBackoffReturnsContextError(t *testing.T) {
	conn, err := newConnection(
		"postgres://u@127.0.0.1:1/db",
		withMaxAttempts(3),
		withBaseDelay(time.Second),
		withMaxDelay(time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := conn.backoff(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("backoff() error = %v, want context.Canceled", err)
	}

	if elapsed := time.Since(start); elapsed >= conn.baseDelay {
		t.Errorf("backoff() waited %v, want it to give up before baseDelay (%v)", elapsed, conn.baseDelay)
	}
}

func TestExecStopsWithoutTrailingBackoff(t *testing.T) {
	conn, err := newConnection(
		"postgres://u@127.0.0.1:1/db",
		withMaxAttempts(2),
		withBaseDelay(100*time.Millisecond),
		withMaxDelay(time.Second),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// The port is closed, so every dial is refused at once and the elapsed time
	// is the sum of the backoffs: connect sleeps 100ms between its two dials and
	// run sleeps 100ms between its two connects, so ~300ms. Sleeping once more
	// before giving up would add baseDelay<<1 = 200ms.
	start := time.Now()
	_, err = conn.Exec(context.Background(), "select 1")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error against a closed port, got nil")
	}

	if elapsed > 400*time.Millisecond {
		t.Errorf("Exec() took %v, want ~300ms: run slept once more before giving up", elapsed)
	}
}

func TestConnectionCloseWithoutSession(t *testing.T) {
	conn, err := newConnection("postgres://u@127.0.0.1:1/db")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ctx := context.Background()

	if err := conn.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	if err := conn.Close(ctx); err != nil {
		t.Fatalf("second Close() error = %v, want nil", err)
	}
}

func TestRawReusesSession(t *testing.T) {
	d := newFakeDialer(t, false)
	conn := newFakeConnection(t, d)
	ctx := context.Background()

	first, err := conn.Raw(ctx)
	if err != nil {
		t.Fatalf("Raw() error = %v", err)
	}

	second, err := conn.Raw(ctx)
	if err != nil {
		t.Fatalf("second Raw() error = %v", err)
	}

	if first != second {
		t.Errorf("Raw() returned a different session, want the cached one")
	}

	if dials, _ := d.counts(); dials != 1 {
		t.Errorf("dials = %d, want 1", dials)
	}
}

func TestRawDialsOnceUnderConcurrency(t *testing.T) {
	d := newFakeDialer(t, true)
	conn := newFakeConnection(t, d)
	ctx := context.Background()

	const goroutines = 20

	sessions := make([]*pgconn.PgConn, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions[i], errs[i] = conn.Raw(ctx)
		}()
	}

	// Every caller is now either dialing or queued on the mutex; give the queued
	// ones time to reach a dial of their own if the lock is not held across it.
	d.waitForDial(1)
	time.Sleep(50 * time.Millisecond)

	if _, overlap := d.counts(); overlap != 1 {
		t.Errorf("overlapping dials = %d, want 1", overlap)
	}

	d.releaseGate()
	wg.Wait()

	for i := range goroutines {
		if errs[i] != nil {
			t.Fatalf("Raw() error = %v", errs[i])
		}
		if sessions[i] != sessions[0] {
			t.Fatalf("goroutine %d got a different session", i)
		}
	}

	if dials, _ := d.counts(); dials != 1 {
		t.Errorf("dials = %d, want 1", dials)
	}
}

func TestRawAfterCloseOpensNewSession(t *testing.T) {
	d := newFakeDialer(t, false)
	conn := newFakeConnection(t, d)
	ctx := context.Background()

	first, err := conn.Raw(ctx)
	if err != nil {
		t.Fatalf("Raw() error = %v", err)
	}

	if err := conn.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !first.IsClosed() {
		t.Error("Close() left the session open")
	}
	if raw := heldSession(conn); raw != nil {
		t.Errorf("held session = %v after Close(), want nil", raw)
	}

	second, err := conn.Raw(ctx)
	if err != nil {
		t.Fatalf("Raw() after Close() error = %v", err)
	}

	if second == first {
		t.Error("Raw() reused the closed session")
	}

	if dials, _ := d.counts(); dials != 2 {
		t.Errorf("dials = %d, want 2", dials)
	}
}

func TestRawReplacesClosedSession(t *testing.T) {
	d := newFakeDialer(t, false)
	conn := newFakeConnection(t, d)
	ctx := context.Background()

	first, err := conn.Raw(ctx)
	if err != nil {
		t.Fatalf("Raw() error = %v", err)
	}

	if err := first.Close(ctx); err != nil {
		t.Fatalf("session Close() error = %v", err)
	}

	second, err := conn.Raw(ctx)
	if err != nil {
		t.Fatalf("Raw() after a dropped session error = %v", err)
	}

	if second == first {
		t.Error("Raw() returned the dropped session")
	}

	if dials, _ := d.counts(); dials != 2 {
		t.Errorf("dials = %d, want 2", dials)
	}
}

func TestCloseWaitsForInFlightDial(t *testing.T) {
	d := newFakeDialer(t, true)
	conn := newFakeConnection(t, d)
	ctx := context.Background()

	acquired := make(chan *pgconn.PgConn, 1)
	go func() {
		session, err := conn.Raw(ctx)
		if err != nil {
			t.Errorf("Raw() error = %v", err)
		}
		acquired <- session
	}()

	d.waitForDial(1)

	var released atomic.Bool
	closed := make(chan error, 1)
	go func() {
		closed <- conn.Close(ctx)
		if !released.Load() {
			t.Error("Close() returned while a dial was still in flight")
		}
	}()

	time.Sleep(50 * time.Millisecond)
	released.Store(true)
	d.releaseGate()

	var session *pgconn.PgConn
	select {
	case session = <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Raw()")
	}

	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Close()")
	}

	if !session.IsClosed() {
		t.Error("Close() left the freshly dialed session open")
	}
	if raw := heldSession(conn); raw != nil {
		t.Errorf("held session = %v after Close(), want nil", raw)
	}
}

func TestAcquireHonoursCancelledContext(t *testing.T) {
	d := newFakeDialer(t, false)
	conn := newFakeConnection(t, d)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := conn.Raw(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Raw() error = %v, want context.Canceled", err)
	}

	if dials, _ := d.counts(); dials != 0 {
		t.Errorf("dials = %d, want 0", dials)
	}
}

func TestConcurrentUseIsRaceFree(t *testing.T) {
	d := newFakeDialer(t, false)
	conn := newFakeConnection(t, d, withMaxAttempts(2), withBaseDelay(time.Millisecond), withMaxDelay(time.Millisecond))
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range 20 {
				switch i % 3 {
				case 0:
					_, _ = conn.Raw(ctx)
				case 1:
					_ = heldSession(conn)
				case 2:
					_ = conn.Close(ctx)
				}
			}
		}()
	}

	wg.Wait()
}

func TestExecReturnsRows(t *testing.T) {
	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) { serveSession(conn, replyRow) }
	conn := newFakeConnection(t, d)

	results, err := conn.Exec(context.Background(), "select 1")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if got := string(results[0].Rows[0][0]); got != "1" {
		t.Errorf("row value = %q, want %q", got, "1")
	}
}

func TestExecLogsNotice(t *testing.T) {
	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) { serveSession(conn, replyNotice) }
	conn := newFakeConnection(t, d)

	notices := make(chan *pgconn.Notice, 1)
	previous := conn.connectionConfig.OnNotice
	conn.connectionConfig.OnNotice = func(c *pgconn.PgConn, n *pgconn.Notice) {
		previous(c, n)
		notices <- n
	}

	if _, err := conn.Exec(context.Background(), "select 1"); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	select {
	case n := <-notices:
		if n.Message != "table already exists" {
			t.Errorf("notice message = %q, want %q", n.Message, "table already exists")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the notice")
	}
}

func TestExecReturnsServerErrorWithoutRedialing(t *testing.T) {
	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) { serveSession(conn, replyError) }
	conn := newFakeConnection(t, d)

	_, err := conn.Exec(context.Background(), "select 1")

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("Exec() error = %v, want a *pgconn.PgError", err)
	}
	if pgErr.Code != "42601" {
		t.Errorf("SQLSTATE = %q, want %q", pgErr.Code, "42601")
	}

	if dials, _ := d.counts(); dials != 1 {
		t.Errorf("dials = %d, want 1: a live session must not be retried", dials)
	}
}

func TestExecRetriesWhenSessionDiesMidCall(t *testing.T) {
	var queries atomic.Int32

	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, dial int) {
		count := func(message pgproto3.FrontendMessage) {
			if _, query := message.(*pgproto3.Query); query {
				queries.Add(1)
			}
		}

		if dial == 1 {
			serveSession(conn, func(_ *pgproto3.Backend, message pgproto3.FrontendMessage) error {
				count(message)

				return errors.New("connection reset")
			})

			return
		}

		serveSession(conn, func(backend *pgproto3.Backend, message pgproto3.FrontendMessage) error {
			count(message)

			return replyRow(backend, message)
		})
	}
	conn := newFakeConnection(t, d, withMaxAttempts(3), withBaseDelay(time.Millisecond), withMaxDelay(time.Millisecond))

	results, err := conn.Exec(context.Background(), "select 1")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	// The dial count cannot be asserted: pgconn dials once more, from another
	// goroutine, to send the cancel request that follows a dropped session.
	if got := queries.Load(); got != 2 {
		t.Errorf("queries served = %d, want 2: the dead session must be replaced", got)
	}
}

func TestExecFailsWhenEverySessionDies(t *testing.T) {
	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) {
		serveSession(conn, func(*pgproto3.Backend, pgproto3.FrontendMessage) error {
			return errors.New("connection reset")
		})
	}
	conn := newFakeConnection(t, d, withMaxAttempts(2), withBaseDelay(time.Millisecond), withMaxDelay(time.Millisecond))

	if _, err := conn.Exec(context.Background(), "select 1"); err == nil {
		t.Fatal("Exec() error = nil, want the exhausted-attempts error")
	} else if !strings.Contains(err.Error(), "exec failed after 2 attempts") {
		t.Errorf("Exec() error = %v, want it to report the attempt count", err)
	}
}

func TestExecBacksOffBetweenDeadSessions(t *testing.T) {
	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) {
		serveSession(conn, func(*pgproto3.Backend, pgproto3.FrontendMessage) error {
			return errors.New("connection reset")
		})
	}
	conn := newFakeConnection(t, d, withMaxAttempts(2), withBaseDelay(100*time.Millisecond), withMaxDelay(time.Second))

	// The fake dialer accepts at once, so connect never sleeps and run's backoff is
	// the only delay: one 100ms wait between the two dead sessions, and none before
	// giving up, which would add baseDelay<<1 = 200ms.
	start := time.Now()
	_, err := conn.Exec(context.Background(), "select 1")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Exec() error = nil, want the exhausted-attempts error")
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("Exec() took %v, want at least one backoff (100ms) between the dead sessions", elapsed)
	}

	if elapsed >= 300*time.Millisecond {
		t.Errorf("Exec() took %v, want under 300ms: run slept once more before giving up", elapsed)
	}
}

func TestExecParamsBindsParameters(t *testing.T) {
	var (
		mu     sync.Mutex
		bound  [][]byte
		oids   []uint32
		binary []int16
	)

	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) {
		serveSession(conn, func(backend *pgproto3.Backend, message pgproto3.FrontendMessage) error {
			switch m := message.(type) {
			case *pgproto3.Parse:
				mu.Lock()
				oids = append([]uint32(nil), m.ParameterOIDs...)
				mu.Unlock()
			case *pgproto3.Bind:
				mu.Lock()
				bound = append([][]byte(nil), m.Parameters...)
				binary = append([]int16(nil), m.ParameterFormatCodes...)
				mu.Unlock()
			}

			return replyRow(backend, message)
		})
	}
	conn := newFakeConnection(t, d)

	params := []param{
		{Value: []byte("7"), OID: 23, Format: 0},
		{Value: []byte("public"), OID: 25, Format: 0},
	}

	result, err := conn.ExecParams(context.Background(), "select $1, $2", params, []int16{0})
	if err != nil {
		t.Fatalf("ExecParams() error = %v", err)
	}

	if len(result.Rows) != 1 {
		t.Fatalf("len(result.Rows) = %d, want 1", len(result.Rows))
	}

	mu.Lock()
	defer mu.Unlock()

	if len(bound) != 2 || string(bound[0]) != "7" || string(bound[1]) != "public" {
		t.Errorf("bound parameters = %q, want [7 public]", bound)
	}
	if len(oids) != 2 || oids[0] != 23 || oids[1] != 25 {
		t.Errorf("parameter OIDs = %v, want [23 25]", oids)
	}
	if len(binary) != 2 || binary[0] != 0 || binary[1] != 0 {
		t.Errorf("parameter formats = %v, want [0 0]", binary)
	}
}

func TestExecParamsReturnsServerError(t *testing.T) {
	d := newFakeDialer(t, false)
	d.serve = func(conn net.Conn, _ int) {
		serveSession(conn, func(backend *pgproto3.Backend, message pgproto3.FrontendMessage) error {
			if _, execute := message.(*pgproto3.Execute); !execute {
				return replyRow(backend, message)
			}

			backend.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: "22012", Message: "division by zero"})

			return nil
		})
	}
	conn := newFakeConnection(t, d)

	result, err := conn.ExecParams(context.Background(), "select 1/$1", []param{{Value: []byte("0"), OID: 23}}, nil)

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("ExecParams() error = %v, want a *pgconn.PgError", err)
	}
	if pgErr.Code != "22012" {
		t.Errorf("SQLSTATE = %q, want %q", pgErr.Code, "22012")
	}
	if result == nil {
		t.Error("ExecParams() result = nil, want the failed attempt's result")
	}
}

func TestConnectStopsWhenContextExpiresDuringBackoff(t *testing.T) {
	d := newFakeDialer(t, true)
	conn := newFakeConnection(t, d, withMaxAttempts(3), withBaseDelay(time.Minute), withMaxDelay(time.Minute))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := conn.Raw(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Raw() error = %v, want context.DeadlineExceeded", err)
	}

	if dials, _ := d.counts(); dials != 1 {
		t.Errorf("dials = %d, want 1: backoff must abort the retry loop", dials)
	}
}

func TestBackoffClampsDelayToMaxDelay(t *testing.T) {
	cases := []struct {
		name    string
		attempt int
	}{
		{name: "shift exceeds max delay", attempt: 4},
		{name: "shift overflows", attempt: 64},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := newConnection(
				"postgres://u@127.0.0.1:1/db",
				withBaseDelay(50*time.Millisecond),
				withMaxDelay(5*time.Millisecond),
			)
			if err != nil {
				t.Fatalf("newConnection() error = %v", err)
			}

			start := time.Now()
			if err := conn.backoff(context.Background(), tt.attempt); err != nil {
				t.Fatalf("backoff() error = %v", err)
			}

			if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
				t.Errorf("backoff() waited %v, want it clamped to maxDelay (%v)", elapsed, conn.maxDelay)
			}
		})
	}
}

package snapshot

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"hump/config"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
	port "hump/internal/port/snapshot"
)

func TestStartValidatesEveryFilterBeforeStarting(t *testing.T) {
	db := newFakeDatabase()
	db.invalid["1; DROP TABLE users"] = true
	r := newTestRunner(t, db, configWith(identifier("users"), identifier("orders")), &fakeProcess{})

	err := r.Start([]port.Request{request("orders", "id > 1"), request("users", "1; DROP TABLE users")})
	if !errors.Is(err, port.ErrInvalidWhere) || !errors.Is(err, database.ErrInvalidWhere) {
		t.Fatalf("Start() error = %v, want ErrInvalidWhere", err)
	}

	if state := currentState(r); state.Status != port.StatusIdle {
		t.Fatalf("status = %s, want idle", state.Status)
	}

	if events := db.snapshotEvents(); len(events) != 0 {
		t.Fatalf("snapshot events = %q, want none", events)
	}
}

func TestStartNamesTheTableWithAnInvalidFilter(t *testing.T) {
	db := newFakeDatabase()
	db.invalid["bad"] = true
	r := newTestRunner(t, db, configWith(), &fakeProcess{})

	tests := []struct {
		table domain.TableIdentifier
		want  string
	}{
		{table: identifier("users"), want: "table public.users: "},
		{table: domain.TableIdentifier{Name: "users"}, want: "table users: "},
	}
	for _, tt := range tests {
		err := r.Start([]port.Request{{Table: tt.table, Where: "bad"}})
		if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
			t.Errorf("Start(%+v) error = %v, want prefix %q", tt.table, err, tt.want)
		}
	}
}

func TestStartRejectsASecondRun(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := r.Start([]port.Request{request("users", "")}); !errors.Is(err, port.ErrAlreadyRunning) {
		t.Fatalf("second Start() error = %v, want ErrAlreadyRunning", err)
	}
}

func TestRunDeliversEveryRowAsInsert(t *testing.T) {
	db := newFakeDatabase()
	db.rows["users"] = []domain.DbRecord{{"id": 1}, {"id": 2}}
	db.rows["orders"] = []domain.DbRecord{{"id": 10}}
	process := &fakeProcess{}
	r := newTestRunner(t, db, configWith(identifier("users"), identifier("orders")), process)

	if err := r.Start([]port.Request{request("users", "id > 0"), request("orders", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	state := waitForState(t, r, isDone)

	for _, table := range state.Tables {
		if table.Status != port.TableSuccess || table.Error != "" {
			t.Errorf("table %s = %s %q, want success", table.Table.Name, table.Status, table.Error)
		}
		if table.CompletedRows != uint64(len(db.rows[string(table.Table.Name)])) {
			t.Errorf("table %s rows = %d, want %d", table.Table.Name, table.CompletedRows, len(db.rows[string(table.Table.Name)]))
		}
		if len(table.Sinks) != 1 || table.Sinks[0].Count != table.CompletedRows {
			t.Errorf("table %s sinks = %+v, want one sink counting every row", table.Table.Name, table.Sinks)
		}
	}

	calls := process.recorded()
	if len(calls) != 3 {
		t.Fatalf("process calls = %d, want 3", len(calls))
	}
	for _, call := range calls {
		if call.operation != domain.OperationTypeInsert || call.prev != nil || call.next == nil || call.dbName != "db" {
			t.Errorf("process call = %+v, want an insert of the row into db", call)
		}
	}

	if events := waitForEvents(t, db, 2); !reflect.DeepEqual(events, []string{"start", "finish"}) {
		t.Errorf("snapshot events = %q, want start then finish", events)
	}

	if want := [][]domain.TableIdentifier{{identifier("users"), identifier("orders")}}; !reflect.DeepEqual(db.started, want) {
		t.Errorf("snapshot tables = %v, want %v", db.started, want)
	}
}

func TestRunSerializesProcessAcrossTables(t *testing.T) {
	db := newFakeDatabase()
	var requests []port.Request
	var tables []domain.TableIdentifier
	for i := range 5 {
		name := domain.TableName(fmt.Sprintf("t%d", i))
		db.rows[string(name)] = []domain.DbRecord{{"id": 1}, {"id": 2}, {"id": 3}}
		requests = append(requests, request(name, ""))
		tables = append(tables, identifier(name))
	}
	process := &fakeProcess{}
	r := newTestRunner(t, db, configWith(tables...), process)

	if err := r.Start(requests); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForState(t, r, isDone)

	if peak := process.peakCalls(); peak != 1 {
		t.Fatalf("peak process calls = %d, want 1", peak)
	}

	if calls := len(process.recorded()); calls != 15 {
		t.Fatalf("process calls = %d, want 15", calls)
	}
}

func TestRunCapsConcurrentReads(t *testing.T) {
	db := newFakeDatabase()
	db.readWait = 5 * time.Millisecond
	var requests []port.Request
	for i := range 6 {
		requests = append(requests, request(domain.TableName(fmt.Sprintf("t%d", i)), ""))
	}
	r := newTestRunner(t, db, configWith(), &fakeProcess{}, WithConcurrency(2))

	if err := r.Start(requests); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForState(t, r, isDone)

	if peak := db.peakReads(); peak != 2 {
		t.Fatalf("peak reads = %d, want 2", peak)
	}
}

func TestRunEstimatesExpectedRows(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	db.counts["users"] = []database.RowCount{{Num: 40, Rough: true}, {Num: 42}}
	db.counts["orders"] = []database.RowCount{{Num: 7, Rough: true}, {Error: errors.New("timeout")}}
	db.countErrs["items"] = errors.New("no such table")
	r := newTestRunner(t, db, configWith(identifier("users"), identifier("orders"), identifier("items")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", ""), request("orders", ""), request("items", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	state := waitForState(t, r, func(state port.State) bool {
		for _, table := range state.Tables {
			if table.Status != port.TableProcessing {
				return false
			}
		}
		return true
	})

	if state.Tables[0].EstimatedRows != 42 || state.Tables[1].EstimatedRows != 7 || state.Tables[2].EstimatedRows != 0 {
		t.Fatalf("estimates = %d, %d, %d, want the exact count, else the rough one, else zero",
			state.Tables[0].EstimatedRows, state.Tables[1].EstimatedRows, state.Tables[2].EstimatedRows)
	}
}

func TestRunReportsAFailedTableAndFinishesTheRest(t *testing.T) {
	db := newFakeDatabase()
	db.rows["users"] = []domain.DbRecord{{"id": 1}}
	db.readErrs["orders"] = errors.New("connection lost")
	r := newTestRunner(t, db, configWith(identifier("users"), identifier("orders")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", ""), request("orders", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	state := waitForState(t, r, isDone)

	if state.Tables[0].Status != port.TableSuccess {
		t.Errorf("users = %s, want success", state.Tables[0].Status)
	}
	if state.Tables[1].Status != port.TableError || state.Tables[1].Error != "connection lost" {
		t.Errorf("orders = %s %q, want error with the read error", state.Tables[1].Status, state.Tables[1].Error)
	}
}

func TestRunFailsATableWhoseRowIsNotProcessed(t *testing.T) {
	db := newFakeDatabase()
	db.rows["users"] = []domain.DbRecord{{"id": 1}, {"id": 2}}
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{err: errors.New("sink down")})

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	state := waitForState(t, r, isDone)

	if table := state.Tables[0]; table.Status != port.TableError || table.Error != "sink down" || table.CompletedRows != 0 {
		t.Fatalf("users = %+v, want an error after no processed rows", table)
	}
}

func TestRunFailsEveryTableWhenTheSnapshotCannotStart(t *testing.T) {
	db := newFakeDatabase()
	db.startErr = errors.New("lock timeout")
	r := newTestRunner(t, db, configWith(identifier("users"), identifier("orders")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", ""), request("orders", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	state := waitForState(t, r, isDone)

	for _, table := range state.Tables {
		if table.Status != port.TableError || table.Error != "lock timeout" {
			t.Errorf("table %s = %s %q, want error with the start error", table.Table.Name, table.Status, table.Error)
		}
	}

	if events := db.snapshotEvents(); !reflect.DeepEqual(events, []string{"start"}) {
		t.Errorf("snapshot events = %q, want no finish for a snapshot that never started", events)
	}
}

func TestRunFinishesWhenTheSnapshotCannotBeReleased(t *testing.T) {
	db := newFakeDatabase()
	db.rows["users"] = []domain.DbRecord{{"id": 1}}
	db.finishErr = errors.New("connection lost")
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if state := waitForState(t, r, isDone); state.Tables[0].Status != port.TableSuccess {
		t.Fatalf("users = %s, want success", state.Tables[0].Status)
	}

	if events := waitForEvents(t, db, 2); !reflect.DeepEqual(events, []string{"start", "finish"}) {
		t.Fatalf("snapshot events = %q, want the release attempted", events)
	}
}

func TestStopFailsUnfinishedTablesAndReleasesTheSnapshot(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForEvents(t, db, 1)

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	state := currentState(r)
	if state.Status != port.StatusDone || state.Tables[0].Status != port.TableError || state.Tables[0].Error != stoppedError {
		t.Fatalf("state = %+v, want done with the table stopped", state)
	}

	if events := waitForEvents(t, db, 2); !reflect.DeepEqual(events, []string{"start", "finish"}) {
		t.Fatalf("snapshot events = %q, want the snapshot released", events)
	}

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() on done error = %v", err)
	}
	if state := currentState(r); state.Status != port.StatusIdle {
		t.Fatalf("status = %s, want idle after clearing", state.Status)
	}

	if err := r.Stop(); !errors.Is(err, port.ErrNotRunning) {
		t.Fatalf("Stop() on idle error = %v, want ErrNotRunning", err)
	}
}

func TestStopSkipsTablesNotYetRead(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	r := newTestRunner(t, db, configWith(), &fakeProcess{}, WithConcurrency(1))

	if err := r.Start([]port.Request{request("users", ""), request("orders", ""), request("items", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForEvents(t, db, 1)

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	waitForEvents(t, db, 2)

	// The second table was already handed to the worker when Stop landed.
	if reads := db.readCount(); reads != 2 {
		t.Fatalf("reads = %d, want 2, the third table skipped", reads)
	}
}

func TestStartAfterStopWaitsForThePreviousSnapshot(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForEvents(t, db, 1)

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}

	if events := waitForEvents(t, db, 3); !reflect.DeepEqual(events[:3], []string{"start", "finish", "start"}) {
		t.Fatalf("snapshot events = %q, want the first snapshot released before the second starts", events)
	}
}

func TestCloseStopsTheRunAndClosesSubscribers(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	_, _, updates := r.Subscribe()

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForEvents(t, db, 1)

	if err := r.onShutdown(t.Context()); err != nil {
		t.Fatalf("onShutdown() error = %v", err)
	}

	if events := db.snapshotEvents(); !reflect.DeepEqual(events, []string{"start", "finish"}) {
		t.Fatalf("snapshot events = %q, want the snapshot released by close", events)
	}

	for range updates {
	}

	if id, _, late := r.Subscribe(); id != 0 {
		t.Fatalf("Subscribe() after close id = %d, want 0", id)
	} else if _, ok := <-late; ok {
		t.Fatal("Subscribe() after close channel is open, want closed")
	}

	if err := r.Start([]port.Request{request("users", "")}); !errors.Is(err, port.ErrClosed) {
		t.Fatalf("Start() after close error = %v, want ErrClosed", err)
	}
}

func TestSubscribeReceivesProgress(t *testing.T) {
	db := newFakeDatabase()
	db.rows["users"] = []domain.DbRecord{{"id": 1}}
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	id, initial, updates := r.Subscribe()
	defer r.Unsubscribe(id)

	if initial.Status != port.StatusIdle {
		t.Fatalf("initial status = %s, want idle", initial.Status)
	}

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	for state := range updates {
		if state.Status == port.StatusDone {
			if state.Tables[0].CompletedRows != 1 {
				t.Fatalf("rows = %d, want 1", state.Tables[0].CompletedRows)
			}
			return
		}
	}

	t.Fatal("updates closed before the run finished")
}

func TestRunningGaugeTracksTheRun(t *testing.T) {
	db := newFakeDatabase()
	db.block = make(chan struct{})
	r := newTestRunner(t, db, configWith(identifier("users")), &fakeProcess{})

	if got := testutil.ToFloat64(r.metrics.running); got != 0 {
		t.Fatalf("running before start = %v, want 0", got)
	}

	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got := testutil.ToFloat64(r.metrics.running); got != 1 {
		t.Fatalf("running during the run = %v, want 1", got)
	}

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got := testutil.ToFloat64(r.metrics.running); got != 0 {
		t.Fatalf("running after stop = %v, want 0", got)
	}

	close(db.block)
	if err := r.Start([]port.Request{request("users", "")}); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	waitForState(t, r, isDone)
	if got := testutil.ToFloat64(r.metrics.running); got != 0 {
		t.Fatalf("running after the run finished = %v, want 0", got)
	}
}

func TestGetSinksKeepsSinksReceivingInserts(t *testing.T) {
	users := identifier("users")
	cfg := configWith(users)
	cfg.Tables[0].Sinks = []sinkcfg.Config{
		{Sink: fakeSink{typ: "kafka", host: "broker", target: "users"}, Operations: []domain.OperationType{domain.OperationTypeInsert}},
		{Sink: fakeSink{typ: "kafka", host: "broker", target: "users"}, Operations: domain.OperationTypes},
		{Sink: fakeSink{typ: "http", host: "api", target: "/deletes"}, Operations: []domain.OperationType{domain.OperationTypeDelete}},
		{Sink: nil, Operations: domain.OperationTypes},
	}

	want := []port.Sink{{Type: "kafka", Host: "broker", Target: "users"}}
	if got := (&Runner{}).getSinks(cfg)[users]; !reflect.DeepEqual(got, want) {
		t.Fatalf("getSinks() = %+v, want %+v", got, want)
	}
}

// fakeDatabase answers the snapshot calls of database.Database from scripted
// rows, recording the order of snapshot events and how many reads overlapped.
type fakeDatabase struct {
	database.Database

	invalid   map[string]bool                // invalid holds the filters ValidateWhere rejects.
	counts    map[string][]database.RowCount // counts are streamed by GetRowCount, keyed by table name.
	countErrs map[string]error               // countErrs fail GetRowCount of a table.
	rows      map[string][]domain.DbRecord   // rows are read by ReadSnapshot, keyed by table name.
	readErrs  map[string]error               // readErrs end ReadSnapshot of a table after its rows.
	startErr  error                          // startErr fails StartSnapshot.
	finishErr error                          // finishErr fails FinishSnapshot.
	block     chan struct{}                  // block, when set, holds every read until closed or ctx is done.
	readWait  time.Duration                  // readWait is how long each read takes before its rows.

	mu      sync.Mutex
	events  []string // events records "start" and "finish" in call order.
	started [][]domain.TableIdentifier
	reads   int
	active  int
	peak    int
}

// newFakeDatabase returns a database with nothing scripted.
func newFakeDatabase() *fakeDatabase {
	return &fakeDatabase{
		invalid:   map[string]bool{},
		counts:    map[string][]database.RowCount{},
		countErrs: map[string]error{},
		rows:      map[string][]domain.DbRecord{},
		readErrs:  map[string]error{},
	}
}

// GetDbName returns a fixed database name.
func (d *fakeDatabase) GetDbName() string {
	return "db"
}

// ValidateWhere rejects the filters in invalid.
func (d *fakeDatabase) ValidateWhere(where string) error {
	if d.invalid[where] {
		return fmt.Errorf("parse where: %w", database.ErrInvalidWhere)
	}

	return nil
}

// GetRowCount streams the counts scripted for the table, or answers its
// scripted error.
func (d *fakeDatabase) GetRowCount(_ context.Context, table domain.TableIdentifier, _ string) (<-chan database.RowCount, error) {
	if err := d.countErrs[string(table.Name)]; err != nil {
		return nil, err
	}

	counts := d.counts[string(table.Name)]

	ch := make(chan database.RowCount, len(counts))
	for _, count := range counts {
		ch <- count
	}
	close(ch)

	return ch, nil
}

// StartSnapshot records the call and answers startErr or a fixed id.
func (d *fakeDatabase) StartSnapshot(_ context.Context, tables []domain.TableIdentifier) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.events = append(d.events, "start")
	d.started = append(d.started, tables)

	if d.startErr != nil {
		return "", d.startErr
	}

	return "snap-1", nil
}

// FinishSnapshot records the call and answers finishErr.
func (d *fakeDatabase) FinishSnapshot(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.events = append(d.events, "finish")

	return d.finishErr
}

// ReadSnapshot hands the rows scripted for the table to onRow, after block and
// readWait, then answers the table's scripted error.
func (d *fakeDatabase) ReadSnapshot(ctx context.Context, snapshotID string, table domain.TableIdentifier, _ string, onRow func(domain.DbRecord) error) error {
	if snapshotID != "snap-1" {
		return fmt.Errorf("unknown snapshot %q", snapshotID)
	}

	d.track(1)
	defer d.track(-1)

	if d.block != nil {
		select {
		case <-d.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	time.Sleep(d.readWait)

	for _, record := range d.rows[string(table.Name)] {
		if err := onRow(record); err != nil {
			return err
		}
	}

	return d.readErrs[string(table.Name)]
}

// track moves the count of reads in flight by delta, keeping its peak and the
// total of reads started.
func (d *fakeDatabase) track(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if delta > 0 {
		d.reads++
	}
	d.active += delta
	d.peak = max(d.peak, d.active)
}

// snapshotEvents returns the recorded snapshot events.
func (d *fakeDatabase) snapshotEvents() []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	return append([]string(nil), d.events...)
}

// readCount returns how many reads were started.
func (d *fakeDatabase) readCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.reads
}

// peakReads returns the most reads that ran at once.
func (d *fakeDatabase) peakReads() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.peak
}

// processCall is one recorded call of the process handler.
type processCall struct {
	operation domain.OperationType
	dbName    domain.DbName
	table     domain.TableIdentifier
	prev      domain.DbRecord
	next      domain.DbRecord
}

// fakeProcess records every call of its handler and how many overlapped.
type fakeProcess struct {
	err error // err is answered by every call.

	mu     sync.Mutex
	calls  []processCall
	active int
	peak   int
}

// handle is the process handler the runner is given.
func (p *fakeProcess) handle(_ context.Context, _ time.Time, operation domain.OperationType, dbName domain.DbName, schema domain.TableSchema, table domain.TableName, _ domain.Headers, prev domain.DbRecord, next domain.DbRecord) error {
	p.mu.Lock()
	p.active++
	p.peak = max(p.peak, p.active)
	p.calls = append(p.calls, processCall{operation: operation, dbName: dbName, table: domain.TableIdentifier{Schema: schema, Name: table}, prev: prev, next: next})
	p.mu.Unlock()

	// Long enough for an unserialized second caller to overlap.
	time.Sleep(time.Millisecond)

	p.mu.Lock()
	p.active--
	p.mu.Unlock()

	return p.err
}

// recorded returns the recorded calls.
func (p *fakeProcess) recorded() []processCall {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]processCall(nil), p.calls...)
}

// peakCalls returns the most calls that ran at once.
func (p *fakeProcess) peakCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.peak
}

// fakeSink is a sink definition rendering a fixed host and target.
type fakeSink struct {
	typ    sink.SinkType
	host   string
	target string
}

// Type returns the scripted sink type.
func (s fakeSink) Type() sink.SinkType { return s.typ }

// DisplayName returns the sink type.
func (s fakeSink) DisplayName() string { return string(s.typ) }

// Metadata returns the scripted host and target.
func (s fakeSink) Metadata(sink.SinkContext) (string, string) { return s.host, s.target }

// configWith returns a configuration of tables, each delivered to a kafka sink
// receiving every operation.
func configWith(tables ...domain.TableIdentifier) config.Configuration {
	cfg := config.Configuration{}
	for _, table := range tables {
		cfg.Tables = append(cfg.Tables, config.Table{
			Schema: table.Schema,
			Name:   table.Name,
			Sinks:  []sinkcfg.Config{{Sink: fakeSink{typ: "kafka", host: "broker", target: string(table.Name)}, Operations: domain.OperationTypes}},
		})
	}

	return cfg
}

// newTestRunner returns a runner over db and process with progress published
// on every change, closed when the test ends.
func newTestRunner(t *testing.T, db *fakeDatabase, cfg config.Configuration, process *fakeProcess, opts ...Option) *Runner {
	t.Helper()

	r := New(db, cfg, process.handle, promauto.With(nil), append([]Option{WithPublishInterval(0)}, opts...)...)
	t.Cleanup(r.close)

	return r
}

// identifier returns the table name in the public schema.
func identifier(name domain.TableName) domain.TableIdentifier {
	return domain.TableIdentifier{Schema: "public", Name: name}
}

// request returns a request for the public table name with where.
func request(name domain.TableName, where string) port.Request {
	return port.Request{Table: identifier(name), Where: where}
}

// currentState returns a copy of the runner's state.
func currentState(r *Runner) port.State {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.clone(r.state)
}

// waitForState polls the runner until ready accepts its state, failing the
// test after five seconds.
func waitForState(t *testing.T, r *Runner, ready func(port.State) bool) port.State {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if state := currentState(r); ready(state) {
			return state
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timed out waiting for state, last %+v", currentState(r))

	return port.State{}
}

// isDone reports whether state is a finished run.
func isDone(state port.State) bool {
	return state.Status == port.StatusDone
}

// waitForEvents polls db until it has recorded n snapshot events, failing the
// test after five seconds.
func waitForEvents(t *testing.T, db *fakeDatabase, n int) []string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if events := db.snapshotEvents(); len(events) >= n {
			return events
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d snapshot events, got %q", n, db.snapshotEvents())

	return nil
}

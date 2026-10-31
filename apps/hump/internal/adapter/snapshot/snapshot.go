// Package snapshot is the snapshot runner: it reads the chosen tables from one
// exported database snapshot and hands every row to the processing pipeline as
// an insert, the way WAL hands over a change, while WAL delivery is paused.
package snapshot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/errgroup"

	"hump/config"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
	port "hump/internal/port/snapshot"
	"hump/pkg/app/shutdown"
	"hump/pkg/messageid"
)

const (
	// defaultConcurrency caps the tables a run reads at once.
	defaultConcurrency = 10
	// defaultPublishInterval is the shortest gap between two progress updates.
	defaultPublishInterval = 200 * time.Millisecond
	// finishTimeout caps releasing the snapshot once a run ends.
	finishTimeout = 10 * time.Second
	// stoppedError is the error a stopped run reports for the tables it did not finish.
	stoppedError = "stopped"
)

// Runner runs one snapshot at a time and publishes its state to every
// subscriber on each change.
type Runner struct {
	db              database.Database
	process         database.ProcessMessageHandler
	sinks           map[domain.TableIdentifier][]port.Sink // sinks are the insert-receiving sinks of each configured table.
	concurrency     int
	publishInterval time.Duration
	metrics         *metrics

	mu          sync.Mutex
	state       port.State
	subs        map[uint64]chan port.State
	nextID      uint64
	cancel      context.CancelFunc // cancel stops the run in progress; nil when none is.
	runDone     chan struct{}      // runDone is closed once the last run released its snapshot; nil before the first run.
	closed      bool
	lastPublish time.Time

	processMu sync.Mutex // processMu serializes process, which is not safe for concurrent use.

	runs sync.WaitGroup // runs tracks the run goroutines, so close can wait for them.
}

// Option configures a Runner.
type Option func(*Runner)

// WithConcurrency sets how many tables a run reads at once. A non-positive n
// leaves the default.
func WithConcurrency(n int) Option {
	return func(r *Runner) {
		if n > 0 {
			r.concurrency = n
		}
	}
}

// WithPublishInterval sets the shortest gap between two progress updates. A
// negative d leaves the default.
func WithPublishInterval(d time.Duration) Option {
	return func(r *Runner) {
		if d >= 0 {
			r.publishInterval = d
		}
	}
}

// New returns an idle runner reading from db and delivering through process,
// registers its running gauge on metricsFactory and its shutdown hook.
func New(db database.Database, cfg config.Configuration, process database.ProcessMessageHandler, metricsFactory promauto.Factory, opts ...Option) *Runner {
	r := &Runner{
		db:              db,
		process:         process,
		concurrency:     defaultConcurrency,
		publishInterval: defaultPublishInterval,
		metrics:         newMetrics(db.GetDbName(), metricsFactory),
		state:           port.State{Status: port.StatusIdle},
		subs:            make(map[uint64]chan port.State),
	}
	r.sinks = r.getSinks(cfg)

	for _, opt := range opts {
		opt(r)
	}

	shutdown.OnShutdown(shutdown.StageWorkers, "snapshot", r.onShutdown)

	return r
}

// Subscribe registers a client and returns its id, the current state and the
// channel later states arrive on.
func (r *Runner) Subscribe() (uint64, port.State, <-chan port.State) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// One slot is enough: every state is complete, so a slow subscriber only
	// needs the latest one.
	ch := make(chan port.State, 1)
	if r.closed {
		close(ch)
		return 0, r.clone(r.state), ch
	}

	r.nextID++
	r.subs[r.nextID] = ch

	return r.nextID, r.clone(r.state), ch
}

// Unsubscribe removes the subscriber with id and closes its channel.
func (r *Runner) Unsubscribe(id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.subs[id]; ok {
		delete(r.subs, id)
		close(ch)
	}
}

// Start validates every filter, then begins a snapshot of tables in the
// background.
func (r *Runner) Start(tables []port.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return port.ErrClosed
	}

	if r.state.Status == port.StatusProcessing {
		return port.ErrAlreadyRunning
	}

	for _, request := range tables {
		if err := r.db.ValidateWhere(request.Where); err != nil {
			return fmt.Errorf("table %s: %w", r.getTableName(request.Table), errors.Join(err, port.ErrInvalidWhere))
		}
	}

	r.state = port.State{
		Status:    port.StatusProcessing,
		StartedAt: time.Now().UTC(),
		Tables:    make([]port.Table, 0, len(tables)),
	}
	for _, request := range tables {
		r.state.Tables = append(r.state.Tables, port.Table{
			Table:  request.Table,
			Where:  request.Where,
			Status: port.TableWait,
			Sinks:  slices.Clone(r.sinks[request.Table]),
		})
	}
	r.metrics.setRunning(true)
	r.notifySubscribers()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	previous := r.runDone
	done := make(chan struct{})
	r.runDone = done

	requests := slices.Clone(tables)
	r.runs.Go(func() {
		defer close(done)

		// A stopped run releases its snapshot in the background; starting before
		// it has would get its id back from the idempotent StartSnapshot.
		if previous != nil {
			<-previous
		}

		r.run(ctx, requests)
	})

	return nil
}

// Stop ends the run in progress, where finished tables succeed and the rest
// fail as stopped, or clears a finished run back to idle.
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch r.state.Status {
	case port.StatusProcessing:
		r.finish()
	case port.StatusDone:
		r.state = port.State{Status: port.StatusIdle}
	default:
		return port.ErrNotRunning
	}
	r.notifySubscribers()

	return nil
}

// run estimates the tables, opens the snapshot and reads the tables from it,
// at most concurrency at once, then finishes the run.
func (r *Runner) run(ctx context.Context, requests []port.Request) {
	if ctx.Err() != nil {
		return
	}

	r.estimate(ctx, requests)

	ids := make([]domain.TableIdentifier, len(requests))
	for i, request := range requests {
		ids[i] = request.Table
	}

	snapshotID, err := r.db.StartSnapshot(ctx, ids)
	if err != nil {
		r.update(ctx, true, func() {
			for i := range r.state.Tables {
				r.state.Tables[i].Status = port.TableDone
				r.state.Tables[i].Error = err.Error()
			}
			r.finish()
		})
		return
	}

	defer func() {
		finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), finishTimeout)
		defer cancel()

		if err := r.db.FinishSnapshot(finishCtx); err != nil {
			logger.Warn("Unable to finish snapshot", "error", err)
		}
	}()

	r.runConcurrently(ctx, len(requests), func(i int) {
		r.readTableSnapshot(ctx, snapshotID, i, requests[i])
	})

	r.update(ctx, true, r.finish)
}

// estimate sets the expected row count of every table: the exact count when
// one arrives, the planner's estimate otherwise. A table that cannot be
// estimated keeps zero.
func (r *Runner) estimate(ctx context.Context, requests []port.Request) {
	r.runConcurrently(ctx, len(requests), func(i int) {
		counts, err := r.db.GetRowCount(ctx, requests[i].Table, requests[i].Where)
		if err != nil {
			return
		}

		var estimate uint64
		for count := range counts {
			if count.Error == nil {
				estimate = count.Num
			}
		}

		r.update(ctx, true, func() {
			r.state.Tables[i].EstimatedRows = estimate
		})
	})
}

// readTableSnapshot reads the table of request i from the snapshot, processing
// every row, and marks it done, carrying the error that ended it, if any.
func (r *Runner) readTableSnapshot(ctx context.Context, snapshotID string, i int, request port.Request) {
	r.update(ctx, true, func() {
		r.state.Tables[i].Status = port.TableProcessing
	})

	dbName := domain.DbName(r.db.GetDbName())

	err := r.db.ReadSnapshot(ctx, snapshotID, request.Table, request.Where, func(record domain.DbRecord, row database.SnapshotRowMetadata) error {
		metadata := database.MessageMetadata{
			MessageID:  messageid.Hash("snapshot", string(dbName), string(request.Table.Schema), string(request.Table.Name), snapshotID, string(row.Schema), string(row.Table), row.CTID),
			SnapshotID: snapshotID,
		}
		r.processMu.Lock()
		err := r.process(ctx, time.Now(), domain.OperationTypeInsert, dbName, request.Table.Schema, request.Table.Name, domain.Headers{}, metadata, nil, record)
		r.processMu.Unlock()
		if err != nil {
			return err
		}

		r.update(ctx, false, func() {
			table := &r.state.Tables[i]
			table.CompletedRows++
			table.EstimatedRows = max(table.EstimatedRows, table.CompletedRows)
			for j := range table.Sinks {
				table.Sinks[j].Count++
			}
		})

		return nil
	})

	r.update(ctx, true, func() {
		r.state.Tables[i].Status = port.TableDone
		if err != nil {
			r.state.Tables[i].Error = err.Error()
		}
	})
}

// runConcurrently calls fn for every index below n, at most concurrency at
// once, and returns once all calls have. Indexes not yet started when ctx ends
// are skipped.
func (r *Runner) runConcurrently(ctx context.Context, n int, fn func(i int)) {
	var group errgroup.Group
	group.SetLimit(r.concurrency)

	for i := range n {
		if ctx.Err() != nil {
			break
		}

		group.Go(func() error {
			fn(i)
			return nil
		})
	}

	_ = group.Wait()
}

// update applies fn to the state and publishes it, unless the run behind ctx
// was stopped, whose state is no longer its own. Without force the publish is
// skipped when the last one is more recent than publishInterval.
func (r *Runner) update(ctx context.Context, force bool, fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop and close cancel under the lock, so an update that raced them sees
	// it here and leaves their state alone.
	if ctx.Err() != nil {
		return
	}

	fn()

	if force || time.Since(r.lastPublish) >= r.publishInterval {
		r.notifySubscribers()
	}
}

// finish stops the run goroutine and turns the state into done: tables done
// without an error succeed, the rest fail. It is called with mu held.
func (r *Runner) finish() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}

	r.state.Status = port.StatusDone
	r.state.FinishedAt = time.Now().UTC()
	r.metrics.setRunning(false)

	for i := range r.state.Tables {
		table := &r.state.Tables[i]
		table.EstimatedRows = 0

		switch {
		case table.Status == port.TableDone && table.Error == "":
			table.Status = port.TableSuccess
		case table.Status == port.TableDone:
			table.Status = port.TableError
		default:
			table.Status = port.TableError
			table.Error = stoppedError
		}
	}
}

// notifySubscribers hands the current state to every subscriber, replacing a
// state the subscriber has not taken yet. It is called with mu held, which is
// what keeps the drain-and-send below free of other senders.
func (r *Runner) notifySubscribers() {
	r.lastPublish = time.Now()

	for _, ch := range r.subs {
		select {
		case <-ch:
		default:
		}

		ch <- r.clone(r.state)
	}
}

// getSinks renders the sinks of every configured table that receive
// inserts, once per distinct sink and target.
func (r *Runner) getSinks(cfg config.Configuration) map[domain.TableIdentifier][]port.Sink {
	result := make(map[domain.TableIdentifier][]port.Sink)

	for _, table := range cfg.ResolveAll() {
		id := domain.TableIdentifier{Schema: table.Schema, Name: table.Name}

		for _, sc := range table.Sinks {
			if sc.Sink == nil || !slices.Contains(sc.Operations, domain.OperationTypeInsert) {
				continue
			}

			host, target := sc.Sink.Metadata(sink.Context{Schema: table.Schema, Table: table.Name})
			entry := port.Sink{Type: string(sc.Sink.Type()), Host: host, Target: target}

			if !slices.Contains(result[id], entry) {
				result[id] = append(result[id], entry)
			}
		}
	}

	return result
}

// getTableName renders table as schema.name, or name alone without a schema.
func (r *Runner) getTableName(table domain.TableIdentifier) string {
	if table.Schema == "" {
		return string(table.Name)
	}

	return string(table.Schema) + "." + string(table.Name)
}

// clone copies state deep enough that the run can keep mutating its own.
func (r *Runner) clone(state port.State) port.State {
	tables := make([]port.Table, len(state.Tables))
	for i, table := range state.Tables {
		table.Sinks = slices.Clone(table.Sinks)
		tables[i] = table
	}
	state.Tables = tables

	return state
}

// onShutdown implements the shutdown hook registered by New.
func (r *Runner) onShutdown(context.Context) error {
	r.close()

	return nil
}

// close stops the run in progress, waits for it to release its snapshot and
// closes every subscriber's channel, so its SSE handler returns. It is
// idempotent.
func (r *Runner) close() {
	r.mu.Lock()
	if !r.closed {
		r.closed = true

		if r.cancel != nil {
			r.cancel()
			r.cancel = nil
		}
		r.metrics.setRunning(false)

		for id, ch := range r.subs {
			delete(r.subs, id)
			close(ch)
		}
	}
	r.mu.Unlock()

	r.runs.Wait()
}

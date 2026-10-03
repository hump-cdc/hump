// Package snapshotmock is a stand-in snapshot runner: it reads no database and
// delivers to no sink, it only plays a plausible run so the web UI has a
// backend to talk to until the real one exists.
package snapshotmock

import (
	"context"
	"hash/fnv"
	"sync"
	"time"

	port "hump/internal/port/snapshot"
	"hump/pkg/app/shutdown"
)

const (
	// defaultTick is how often a run advances and publishes its state.
	defaultTick = 400 * time.Millisecond
	// concurrentTables caps the tables a run reads at once.
	concurrentTables = 2
	// rowsPerTick is how many rows a table being read advances per tick.
	rowsPerTick = 600
	// stoppedError is the error a stopped run reports for the tables it did not finish.
	stoppedError = "stopped"
)

// Runner plays one fake snapshot at a time and publishes its state to every
// subscriber on each change.
type Runner struct {
	tick time.Duration

	mu     sync.Mutex
	state  port.State
	subs   map[uint64]chan port.State
	nextID uint64
	cancel context.CancelFunc // cancel stops the run in progress; nil when none is.
	closed bool

	runs sync.WaitGroup // runs tracks the run goroutines, so close can wait for them.
}

// Option configures a Runner.
type Option func(*Runner)

// WithTick sets how often a run advances. A non-positive d leaves the default.
func WithTick(d time.Duration) Option {
	return func(r *Runner) {
		if d > 0 {
			r.tick = d
		}
	}
}

// New returns an idle runner and registers its shutdown hook.
func New(opts ...Option) *Runner {
	r := &Runner{
		tick:  defaultTick,
		state: port.State{Status: port.StatusIdle},
		subs:  make(map[uint64]chan port.State),
	}

	for _, opt := range opts {
		opt(r)
	}

	shutdown.OnShutdown(shutdown.StageWorkers, "snapshot_mock", r.onShutdown)

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
		return 0, clone(r.state), ch
	}

	r.nextID++
	r.subs[r.nextID] = ch

	return r.nextID, clone(r.state), ch
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

// Start begins a fake snapshot of tables. The filters are ignored.
func (r *Runner) Start(tables []port.Request) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state.Status == port.StatusProcessing {
		return port.ErrAlreadyRunning
	}

	if r.closed {
		return port.ErrNotRunning
	}

	r.state = port.State{
		Status:    port.StatusProcessing,
		StartedAt: time.Now().UTC(),
		Tables:    make([]port.Table, 0, len(tables)),
	}
	for _, request := range tables {
		r.state.Tables = append(r.state.Tables, mockTable(string(request.Table.Schema), string(request.Table.Name), request.Where))
	}
	r.publish()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.runs.Go(func() { r.run(ctx) })

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
	r.publish()

	return nil
}

// run advances the run every tick until it finishes or ctx is cancelled.
func (r *Runner) run(ctx context.Context) {
	ticker := time.NewTicker(r.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			r.mu.Lock()
			// Stop and close cancel under the lock, so a tick that raced them
			// sees it here and leaves their state alone.
			if ctx.Err() != nil {
				r.mu.Unlock()
				return
			}

			if r.advance() {
				r.finish()
			}
			r.publish()
			r.mu.Unlock()
		}
	}
}

// advance moves the run one tick forward and reports whether every table is
// done. It is called with mu held.
func (r *Runner) advance() bool {
	active := 0
	for i := range r.state.Tables {
		if r.state.Tables[i].Status == port.TableProcessing {
			active++
		}
	}

	finished := true
	for i := range r.state.Tables {
		table := &r.state.Tables[i]

		if table.Status == port.TableWait && active < concurrentTables {
			table.Status = port.TableProcessing
			active++
		}

		if table.Status == port.TableProcessing {
			table.Rows = min(table.Rows+rowsPerTick, table.EstimatedRows)
			for j := range table.Sinks {
				table.Sinks[j].Count = table.Rows
			}

			if table.Rows == table.EstimatedRows {
				table.Status = port.TableDone
			}
		}

		if table.Status != port.TableDone {
			finished = false
		}
	}

	return finished
}

// finish stops the run goroutine and turns the state into done. It is called
// with mu held.
func (r *Runner) finish() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}

	r.state.Status = port.StatusDone
	r.state.FinishedAt = time.Now().UTC()

	for i := range r.state.Tables {
		table := &r.state.Tables[i]
		table.EstimatedRows = 0

		if table.Status == port.TableDone {
			table.Status = port.TableSuccess
		} else {
			table.Status = port.TableError
			table.Error = stoppedError
		}
	}
}

// publish hands the current state to every subscriber, replacing a state the
// subscriber has not taken yet. It is called with mu held, which is what keeps
// the drain-and-send below free of other senders.
func (r *Runner) publish() {
	for _, ch := range r.subs {
		select {
		case <-ch:
		default:
		}

		ch <- clone(r.state)
	}
}

// onShutdown implements the shutdown hook registered by New.
func (r *Runner) onShutdown(context.Context) error {
	r.close()

	return nil
}

// close stops the run in progress and closes every subscriber's channel, so
// its SSE handler returns. It is idempotent.
func (r *Runner) close() {
	r.mu.Lock()
	if !r.closed {
		r.closed = true

		if r.cancel != nil {
			r.cancel()
			r.cancel = nil
		}

		for id, ch := range r.subs {
			delete(r.subs, id)
			close(ch)
		}
	}
	r.mu.Unlock()

	r.runs.Wait()
}

// mockTable is a table about to be read, with a row count and sinks derived
// from its name, so the same table looks the same on every run.
func mockTable(schema, name, where string) port.Table {
	h := fnv.New32a()
	_, _ = h.Write([]byte(schema + "." + name))
	seed := h.Sum32()

	sinks := []port.Sink{{Type: "nats-core", Host: "localhost:4222", Target: "hump." + schema + "." + name}}
	if seed%2 == 0 {
		sinks = append(sinks, port.Sink{Type: "kafka", Host: "localhost:9092", Target: schema + "." + name})
	}

	return port.Table{
		Schema:        schema,
		Name:          name,
		Where:         where,
		Status:        port.TableWait,
		EstimatedRows: 5_000 + uint64(seed%195_000),
		Sinks:         sinks,
	}
}

// clone copies state deep enough that the run can keep mutating its own.
func clone(state port.State) port.State {
	tables := make([]port.Table, len(state.Tables))
	for i, table := range state.Tables {
		table.Sinks = append([]port.Sink(nil), table.Sinks...)
		tables[i] = table
	}
	state.Tables = tables

	return state
}

package process_operation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"hump/config"
	cfgresilience "hump/config/resilience"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	workerpool "hump/pkg/worker_pool"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func TestUseCaseProcessDeliversToAcceptingSinks(t *testing.T) {
	sinks := &recordingSinks{}
	uc := newTestUseCase(t, sinks,
		config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
			newSinkConfig("inserts", domain.OperationTypeInsert),
			newSinkConfig("updates", domain.OperationTypeUpdate),
		}},
		config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
			newSinkConfig("all", domain.OperationTypeInsert, domain.OperationTypeUpdate),
		}},
	)

	err := NewController(uc).Process(context.Background(), time.Now(), domain.OperationTypeInsert,
		"app", "public", "users", domain.Headers{"traceparent": "00-abc"}, database.MessageMetadata{MessageID: "event-1", SnapshotID: "snapshot-1"}, nil, domain.DbRecord{"id": 1})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if got, want := sinks.delivered(), []string{"all", "inserts"}; !slices.Equal(got, want) {
		t.Fatalf("delivered to %v, want %v", got, want)
	}
	for _, row := range sinks.contexts {
		if row.MessageId != "event-1" || row.SnapshotID != "snapshot-1" {
			t.Fatalf("identity not forwarded: %+v", row)
		}
	}
}

func TestUseCaseProcessUnknownTable(t *testing.T) {
	sinks := &recordingSinks{}
	uc := newTestUseCase(t, sinks, config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
		newSinkConfig("users", domain.OperationTypeInsert),
	}})

	if err := process(context.Background(), uc, "orders"); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if got := sinks.delivered(); len(got) != 0 {
		t.Fatalf("delivered to %v, want none", got)
	}
}

func TestUseCaseProcessReportsFailedSink(t *testing.T) {
	sinks := &recordingSinks{err: errBoom}
	uc := newTestUseCase(t, sinks, config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
		newSinkConfig("users", domain.OperationTypeInsert),
	}})

	if err := process(context.Background(), uc, "users"); !errors.Is(err, workerpool.ErrWorkerFailed) {
		t.Fatalf("Process() error = %v, want %v", err, workerpool.ErrWorkerFailed)
	}

	if err := uc.wp.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	awaitErrorsDrained(t, uc)
}

func TestUseCaseProcessReportsSubmitError(t *testing.T) {
	sinks := &recordingSinks{}
	uc := newTestUseCase(t, sinks, config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
		newSinkConfig("users", domain.OperationTypeInsert),
	}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := process(ctx, uc, "users"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Process() error = %v, want %v", err, context.Canceled)
	}
	if got := sinks.delivered(); len(got) != 0 {
		t.Fatalf("delivered to %v, want none", got)
	}
}

func TestUseCaseProcessAfterShutdown(t *testing.T) {
	uc := newTestUseCase(t, &recordingSinks{}, config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
		newSinkConfig("users", domain.OperationTypeInsert),
	}})

	if err := uc.wp.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if err := process(context.Background(), uc, "users"); !errors.Is(err, workerpool.ErrWorkerPoolClosed) {
		t.Fatalf("Process() error = %v, want %v", err, workerpool.ErrWorkerPoolClosed)
	}
}

func TestUseCaseShutdownHookClosesPool(t *testing.T) {
	uc := newTestUseCase(t, &recordingSinks{}, config.Table{Schema: "public", Name: "users", Sinks: []sinkcfg.Config{
		newSinkConfig("users", domain.OperationTypeInsert),
	}})

	// The registry is process-global and runs once: pools of earlier tests are
	// already closed and report so, and a repeated run (-count) finds it spent.
	_ = shutdown.Shutdown(time.Second)

	err := process(context.Background(), uc, "users")
	if err == nil {
		t.Skip("the global shutdown already ran in this process")
	}
	if !errors.Is(err, workerpool.ErrWorkerPoolClosed) {
		t.Fatalf("Process() error = %v, want %v", err, workerpool.ErrWorkerPoolClosed)
	}
}

func TestNewUseCaseWithoutSinks(t *testing.T) {
	for name, tbls := range map[string][]config.Table{
		"no tables":      nil,
		"table no sinks": {{Schema: "public", Name: "users"}},
	} {
		t.Run(name, func(t *testing.T) {
			sinks := &recordingSinks{}
			uc := newTestUseCase(t, sinks, tbls...)

			if err := process(context.Background(), uc, "users"); err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if got := sinks.delivered(); len(got) != 0 {
				t.Fatalf("delivered to %v, want none", got)
			}
		})
	}
}

func TestProcessLogsExcludeRowContents(t *testing.T) {
	if os.Getenv("HUMP_PROCESS_LOG_CHILD") == "1" {
		for _, outcome := range []string{"success", "failure", "cancelled"} {
			processLogProbe(t, outcome)
		}
		return
	}

	for _, verbose := range []string{"false", "true"} {
		t.Run("verbose="+verbose, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestProcessLogsExcludeRowContents$")
			cmd.Env = append(os.Environ(), "HUMP_PROCESS_LOG_CHILD=1", "LOG_FORMAT=json", "LOG_VERBOSE="+verbose)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("child: %v\n%s", err, out)
			}
			for _, forbidden := range []string{"sentinel-old-row", "sentinel-new-row", `"prev":`, `"next":`, "errror"} {
				if strings.Contains(string(out), forbidden) {
					t.Errorf("logs contain %q: %s", forbidden, out)
				}
			}

			messages := map[string]int{}
			for _, line := range strings.Split(string(out), "\n") {
				if !strings.HasPrefix(line, "{") {
					continue
				}
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				messages[record["msg"].(string)]++
				for key, want := range map[string]string{
					"operation": "update", "database": "source", "schema": "public", "table": "users",
					"trace_id": "4bf92f3577b34da6a3ce929d0e0e4736", "span_id": "00f067aa0ba902b7",
				} {
					if record[key] != want {
						t.Errorf("%s = %v, want %q", key, record[key], want)
					}
				}
				if strings.Count(line, `"operation":`) != 1 {
					t.Errorf("duplicate operation key: %s", line)
				}
				if record["level"] == "ERROR" && record["error"] == nil {
					t.Errorf("error missing: %s", line)
				}
			}
			for _, message := range []string{"Failed to process", "Failed to submit task to worker pool"} {
				if messages[message] != 1 {
					t.Errorf("%q count = %d, want 1", message, messages[message])
				}
			}
			wantProcessing, wantProcessed := 0, 0
			if verbose == "true" {
				wantProcessing, wantProcessed = 3, 1
			}
			if messages["Processing"] != wantProcessing || messages["Processed"] != wantProcessed {
				t.Errorf("debug messages = %v", messages)
			}
		})
	}
}

// tables is a [Configuration] over a fixed list of tables.
type tables []config.Table

// ResolveAll implements [Configuration].
func (t tables) ResolveAll() []config.Table { return t }

// namedDefinition is a sink definition identified by its name.
type namedDefinition string

func (namedDefinition) Type() sink.Type       { return sink.SinkTypeKafka }
func (d namedDefinition) DisplayName() string { return string(d) }
func (d namedDefinition) Metadata(sink.Context) (string, string) {
	return "host", string(d)
}

// recordingSinks records the name of every definition it delivers to and fails
// each delivery with err.
type recordingSinks struct {
	err      error
	mu       sync.Mutex
	names    []string
	contexts []sink.Context
}

// Process implements [sink.Sink].
func (s *recordingSinks) Process(_ context.Context, definition sink.Definition, row *sink.Context, _ *sink.Capture) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.names = append(s.names, definition.DisplayName())
	s.contexts = append(s.contexts, *row)

	return s.err
}

// delivered returns the sorted names of the definitions delivered to.
func (s *recordingSinks) delivered() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Sorted(slices.Values(s.names))
}

// newTestUseCase builds a use case over tbls delivering to sinks, shutting its
// worker pool down when the test ends.
func newTestUseCase(t *testing.T, sinks sink.Sink, tbls ...config.Table) *UseCase {
	t.Helper()

	uc := NewUseCase(tables(tbls), "app", sinks, promauto.With(prometheus.NewRegistry()), nil, nil)
	t.Cleanup(func() { _ = uc.wp.Shutdown() })

	return uc
}

// newSinkConfig returns a single-attempt, crashing sink config named name that
// receives ops.
func newSinkConfig(name string, ops ...domain.OperationType) sinkcfg.Config {
	cfg := sinkcfg.Config{
		Sink:       namedDefinition(name),
		Failure:    &cfgresilience.Failure{},
		Retry:      &cfgresilience.Retry{MaxAttempts: 1, AttemptTimeout: time.Second},
		Operations: ops,
	}
	cfg.Failure.SetDefaults()

	return cfg
}

// process delivers an insert of one row of public.table.
func process(ctx context.Context, uc *UseCase, table domain.TableName) error {
	return uc.Process(ctx, time.Now(), domain.OperationTypeInsert, "app", "public", table, nil, database.MessageMetadata{}, nil, domain.DbRecord{"id": 1})
}

// awaitErrorsDrained waits until the use case's logging goroutine has taken
// every worker error off the closed pool.
func awaitErrorsDrained(t *testing.T, uc *UseCase) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for len(uc.wp.Errors()) > 0 {
		if time.Now().After(deadline) {
			t.Fatal("worker errors were not drained")
		}
		time.Sleep(time.Millisecond)
	}
}

// processLogProbe exercises the real batch and logging paths without sink adapters.
func processLogProbe(t *testing.T, outcome string) {
	t.Helper()
	wp := workerpool.New[*sink.Context](workerpool.WithWorkers(1), workerpool.WithQueueSize(1))
	defer func() {
		if err := wp.Shutdown(); err != nil {
			t.Error(err)
		}
	}()
	uc := &UseCase{
		wp: wp,
		bindings: map[domain.TableIdentifier][]*binding{
			{Schema: "public", Name: "users"}: {{
				operations: map[domain.OperationType]struct{}{domain.OperationTypeUpdate: {}},
				fn: func(_ context.Context, row *sink.Context) error {
					if row.Prev["secret"] != "sentinel-old-row" || row.Next["secret"] != "sentinel-new-row" {
						return errors.New("row contents changed")
					}
					if outcome == "failure" {
						return errors.New("delivery failed")
					}
					return nil
				},
			}},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if outcome == "cancelled" {
		cancel()
	}
	err := uc.Process(ctx, time.Now(), domain.OperationTypeUpdate, "source", "public", "users",
		domain.Headers{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},
		database.MessageMetadata{},
		domain.DbRecord{"secret": "sentinel-old-row"}, domain.DbRecord{"secret": "sentinel-new-row"})
	if (err != nil) != (outcome != "success") {
		t.Fatalf("%s: error = %v", outcome, err)
	}
}

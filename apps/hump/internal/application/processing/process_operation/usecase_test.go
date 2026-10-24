package process_operation

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"hump/config"
	cfgresilience "hump/config/resilience"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
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
		"app", "public", "users", domain.Headers{"traceparent": "00-abc"}, nil, domain.DbRecord{"id": 1})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if got, want := sinks.delivered(), []string{"all", "inserts"}; !slices.Equal(got, want) {
		t.Fatalf("delivered to %v, want %v", got, want)
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
	err   error
	mu    sync.Mutex
	names []string
}

// Process implements [sink.Sink].
func (s *recordingSinks) Process(_ context.Context, definition sink.Definition, _ *sink.Context, _ *sink.Capture) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.names = append(s.names, definition.DisplayName())

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
	return uc.Process(ctx, time.Now(), domain.OperationTypeInsert, "app", "public", table, nil, nil, domain.DbRecord{"id": 1})
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

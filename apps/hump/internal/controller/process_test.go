package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hump/config"
	cfgresilience "hump/config/resilience"
	sinkcfg "hump/config/sink"
	"hump/internal/application/processing/process_operation"
	"hump/internal/domain"
	"hump/internal/port/database"
	"hump/internal/port/sink"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func TestProcessControllerDeliversListenedRows(t *testing.T) {
	sinks := &countingSinks{}
	db := &fakeListener{}
	uc := process_operation.NewUseCase(processConfig{}, "app", sinks, promauto.With(prometheus.NewRegistry()), nil, nil)

	if err := NewProcessController(db, process_operation.NewController(uc)).Listen(context.Background()); err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	if got := sinks.count(); got != 1 {
		t.Fatalf("sink received %d rows, want the one row the database streamed", got)
	}
}

func TestProcessControllerReturnsTheListenError(t *testing.T) {
	boom := errors.New("replication slot is gone")
	db := &fakeListener{err: boom}
	uc := process_operation.NewUseCase(processConfig{}, "app", &countingSinks{}, promauto.With(prometheus.NewRegistry()), nil, nil)

	if err := NewProcessController(db, process_operation.NewController(uc)).Listen(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("Listen() error = %v, want %v", err, boom)
	}
}

// fakeListener streams one insert into public.users to the handler, then
// returns err.
type fakeListener struct {
	database.Database

	err error
}

// Listen implements [database.Database].
func (l *fakeListener) Listen(ctx context.Context, handler database.ProcessMessageHandler) error {
	if l.err != nil {
		return l.err
	}

	return handler(ctx, time.Now(), domain.OperationTypeInsert, "app", "public", "users", nil, nil, domain.DbRecord{"id": 1})
}

// processConfig configures public.users with one single-attempt sink.
type processConfig struct{}

// ResolveAll implements [process_operation.Configuration].
func (processConfig) ResolveAll() []config.Table {
	failure := &cfgresilience.Failure{}
	failure.SetDefaults()

	return []config.Table{{
		Schema: "public",
		Name:   "users",
		Sinks: []sinkcfg.Config{{
			Sink:       testDefinition{},
			Failure:    failure,
			Retry:      &cfgresilience.Retry{MaxAttempts: 1, AttemptTimeout: time.Second},
			Operations: []domain.OperationType{domain.OperationTypeInsert},
		}},
	}}
}

// testDefinition is a sink definition with a fixed host and target.
type testDefinition struct{}

func (testDefinition) Type() sink.Type                        { return sink.SinkTypeKafka }
func (testDefinition) DisplayName() string                    { return "test" }
func (testDefinition) Metadata(sink.Context) (string, string) { return "host", "users" }

// countingSinks counts the rows delivered to it.
type countingSinks struct {
	mu   sync.Mutex
	rows int
}

// Process implements [sink.Sink].
func (s *countingSinks) Process(context.Context, sink.Definition, *sink.Context, *sink.Capture) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rows++

	return nil
}

// count returns the number of rows delivered so far.
func (s *countingSinks) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rows
}

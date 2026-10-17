package shutdown

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestShutdownRunsStagesInOrder(t *testing.T) {
	r := &registry{}
	rec := &orderRecorder{}

	r.add(StageServers, "web", rec.handler("web"))
	r.add(StageSinks, "kafka", rec.handler("kafka"))
	r.add(StageWorkers, "pool", rec.handler("pool"))
	r.add(StageSinks, "stream_env", rec.handler("stream_env"))
	r.add(StageSinks, "stream_producer", rec.handler("stream_producer"))
	r.add(StageServers, "observability", rec.handler("observability"))

	if err := r.shutdown(time.Second); err != nil {
		t.Fatalf("shutdown() error = %v, want nil", err)
	}

	want := []string{"pool", "stream_producer", "stream_env", "kafka", "observability", "web"}
	if got := rec.recorded(); !slices.Equal(got, want) {
		t.Errorf("handlers ran in order %v, want %v", got, want)
	}
}

func TestShutdownRunsOnce(t *testing.T) {
	r := &registry{}
	rec := &orderRecorder{}

	r.add(StageWorkers, "pool", rec.handler("pool"))

	_ = r.shutdown(time.Second)
	_ = r.shutdown(time.Second)

	if got := rec.recorded(); len(got) != 1 {
		t.Errorf("handler ran %d times, want 1", len(got))
	}
}

func TestShutdownContinuesAfterTimeout(t *testing.T) {
	r := &registry{}
	rec := &orderRecorder{}

	block := make(chan struct{})
	t.Cleanup(func() { close(block) })

	r.add(StageServers, "web", rec.handler("web"))
	r.add(StageSinks, "stuck", func(context.Context) error {
		<-block
		return nil
	})

	if err := r.shutdown(10 * time.Millisecond); !errors.Is(err, ErrShutdownTimeout) {
		t.Fatalf("shutdown() error = %v, want ErrShutdownTimeout", err)
	}

	if got := rec.recorded(); !slices.Equal(got, []string{"web"}) {
		t.Errorf("handlers ran %v, want [web]", got)
	}
}

func TestShutdownWithoutHandlers(t *testing.T) {
	r := &registry{}

	if err := r.shutdown(time.Second); err != nil {
		t.Fatalf("shutdown() error = %v, want nil", err)
	}
}

func TestShutdownContinuesAfterHandlerError(t *testing.T) {
	r := &registry{}
	rec := &orderRecorder{}

	wantErr := errors.New("close failed")
	r.add(StageServers, "web", rec.handler("web"))
	r.add(StageSinks, "kafka", func(context.Context) error {
		return wantErr
	})

	if err := r.shutdown(time.Second); !errors.Is(err, wantErr) {
		t.Fatalf("shutdown() error = %v, want %v", err, wantErr)
	}

	if got := rec.recorded(); !slices.Equal(got, []string{"web"}) {
		t.Errorf("handlers ran %v, want [web]", got)
	}
}

func TestGlobalShutdownRunsRegisteredHandlers(t *testing.T) {
	oldRegistry := globalRegistry
	globalRegistry = &registry{}
	t.Cleanup(func() { globalRegistry = oldRegistry })

	rec := &orderRecorder{}

	OnShutdown(StageWorkers, "pool", rec.handler("pool"))

	if err := Shutdown(time.Second); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}

	if got := rec.recorded(); !slices.Equal(got, []string{"pool"}) {
		t.Errorf("handlers ran %v, want [pool]", got)
	}
}

func TestDrainClosesDraining(t *testing.T) {
	Drain()
	Drain()

	select {
	case <-Draining():
	default:
		t.Fatal("Draining() is open after Drain()")
	}
}

// orderRecorder records the names of the handlers it built, in the order they ran.
type orderRecorder struct {
	mu    sync.Mutex
	names []string
}

// handler returns a shutdown handler that records name.
func (r *orderRecorder) handler(name string) func(context.Context) error {
	return func(context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()

		r.names = append(r.names, name)

		return nil
	}
}

// recorded returns the recorded names.
func (r *orderRecorder) recorded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.names)
}

package nats

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	sinkcfg "hump/config/sink"
)

func TestEnsureReconciledSkipsWithoutSubjects(t *testing.T) {
	js := &fakeJS{}

	if err := newReconciler(js).ensureReconciled(context.Background(), sinkcfg.NatsJetStreamSink{}); err != nil {
		t.Fatalf("ensureReconciled() error = %v", err)
	}
	if len(js.calls) != 0 {
		t.Errorf("calls = %v, want none", js.calls)
	}
}

func TestEnsureReconciledCreateOrUpdateBuildsConfig(t *testing.T) {
	tests := []struct {
		retention, storage string
		wantRetention      jetstream.RetentionPolicy
		wantStorage        jetstream.StorageType
	}{
		{"", "", jetstream.LimitsPolicy, jetstream.FileStorage},
		{"limits", "file", jetstream.LimitsPolicy, jetstream.FileStorage},
		{"interest", "memory", jetstream.InterestPolicy, jetstream.MemoryStorage},
		{"workqueue", "memory", jetstream.WorkQueuePolicy, jetstream.MemoryStorage},
	}
	for _, tt := range tests {
		t.Run(tt.retention+"/"+tt.storage, func(t *testing.T) {
			js := &fakeJS{}
			cfg := jetStreamSink(sinkcfg.JetStreamReconcileCreateOrUpdate)
			cfg.Retention, cfg.Storage = tt.retention, tt.storage

			if err := newReconciler(js).ensureReconciled(context.Background(), cfg); err != nil {
				t.Fatalf("ensureReconciled() error = %v", err)
			}

			if len(js.calls) != 1 || js.calls[0] != "CreateOrUpdateStream" {
				t.Fatalf("calls = %v, want CreateOrUpdateStream", js.calls)
			}
			got := js.config
			if got.Name != "orders" || len(got.Subjects) != 1 || got.Subjects[0] != "orders.>" {
				t.Errorf("config = %+v, want orders with subject orders.>", got)
			}
			if got.Retention != tt.wantRetention || got.Storage != tt.wantStorage {
				t.Errorf("retention, storage = %v, %v, want %v, %v", got.Retention, got.Storage, tt.wantRetention, tt.wantStorage)
			}
		})
	}
}

func TestEnsureReconciledCreateWhenMissing(t *testing.T) {
	tests := []struct {
		name      string
		js        *fakeJS
		wantCalls []string
		wantErr   error
	}{
		{"stream exists", &fakeJS{}, []string{"Stream"}, nil},
		{"stream missing", &fakeJS{streamErr: jetstream.ErrStreamNotFound}, []string{"Stream", "CreateStream"}, nil},
		{"create fails", &fakeJS{streamErr: jetstream.ErrStreamNotFound, createErr: jetstream.ErrStreamNameAlreadyInUse}, []string{"Stream", "CreateStream"}, errReconcile},
		{"lookup fails", &fakeJS{streamErr: jetstream.ErrJetStreamNotEnabled}, []string{"Stream"}, errReconcile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newReconciler(tt.js).ensureReconciled(context.Background(), jetStreamSink(sinkcfg.JetStreamReconcileCreateWhenMissing))
			if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil && err != nil) {
				t.Fatalf("ensureReconciled() error = %v, want %v", err, tt.wantErr)
			}
			if len(tt.js.calls) != len(tt.wantCalls) {
				t.Fatalf("calls = %v, want %v", tt.js.calls, tt.wantCalls)
			}
			for i := range tt.wantCalls {
				if tt.js.calls[i] != tt.wantCalls[i] {
					t.Errorf("calls = %v, want %v", tt.js.calls, tt.wantCalls)
				}
			}
		})
	}
}

func TestEnsureReconciledCachesStickyOutcome(t *testing.T) {
	js := &fakeJS{createOrUpdateErr: errors.New("insufficient resources")}
	r := newReconciler(js)
	cfg := jetStreamSink(sinkcfg.JetStreamReconcileCreateOrUpdate)

	first := r.ensureReconciled(context.Background(), cfg)
	second := r.ensureReconciled(context.Background(), cfg)

	if !errors.Is(first, errReconcile) || !errors.Is(second, errReconcile) {
		t.Fatalf("errors = %v, %v, want errReconcile twice", first, second)
	}
	if len(js.calls) != 1 {
		t.Errorf("calls = %v, want one reconcile", js.calls)
	}
}

func TestEnsureReconciledRetriesTransientFailure(t *testing.T) {
	js := &fakeJS{createOrUpdateErr: nats.ErrTimeout}
	r := newReconciler(js)
	cfg := jetStreamSink(sinkcfg.JetStreamReconcileCreateOrUpdate)

	if err := r.ensureReconciled(context.Background(), cfg); !errors.Is(err, nats.ErrTimeout) || errors.Is(err, errReconcile) {
		t.Fatalf("ensureReconciled() error = %v, want raw ErrTimeout", err)
	}

	js.createOrUpdateErr = nil
	if err := r.ensureReconciled(context.Background(), cfg); err != nil {
		t.Fatalf("ensureReconciled() retry error = %v", err)
	}
	if len(js.calls) != 2 {
		t.Errorf("calls = %v, want the transient failure retried", js.calls)
	}
}

// fakeJS is a jetstream.JetStream recording the stream management calls the
// reconciler makes; the embedded nil interface panics on anything else.
type fakeJS struct {
	jetstream.JetStream

	streamErr         error
	createErr         error
	createOrUpdateErr error

	calls  []string
	config jetstream.StreamConfig
}

// Stream records the lookup and returns streamErr.
func (f *fakeJS) Stream(context.Context, string) (jetstream.Stream, error) {
	f.calls = append(f.calls, "Stream")
	return nil, f.streamErr
}

// CreateStream records cfg and returns createErr.
func (f *fakeJS) CreateStream(_ context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	f.calls = append(f.calls, "CreateStream")
	f.config = cfg
	return nil, f.createErr
}

// CreateOrUpdateStream records cfg and returns createOrUpdateErr.
func (f *fakeJS) CreateOrUpdateStream(_ context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	f.calls = append(f.calls, "CreateOrUpdateStream")
	f.config = cfg
	return nil, f.createOrUpdateErr
}

// jetStreamSink returns a JetStream sink for stream "orders" on subject
// "orders.>" reconciled in mode.
func jetStreamSink(mode sinkcfg.JetStreamReconcileMode) sinkcfg.NatsJetStreamSink {
	return sinkcfg.NatsJetStreamSink{
		Stream: sinkcfg.NatsStream{Name: "orders", Subjects: []string{"orders.>"}},
		Mode:   mode,
	}
}

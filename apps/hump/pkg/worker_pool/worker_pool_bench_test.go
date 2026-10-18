package workerpool

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

var sinkCounts = []int{1, 2, 5, 10, 50, 100}

var taskDurations = []struct {
	name string
	d    time.Duration
}{
	{"0.0ms", 0},
	{"0.1ms", 100 * time.Microsecond},
	{"0.5ms", 500 * time.Microsecond},
	{"1.5ms", 1500 * time.Microsecond},
	{"10.0ms", 10000 * time.Microsecond},
}

type benchPayload struct {
	Version string

	SinkType string
	SinkName string

	Timestamp time.Time
	Operation string

	Database string
	Schema   string
	Table    string

	Headers map[string]string

	Prev map[string]any
	Next map[string]any
}

func BenchmarkWorkerPool(b *testing.B) {
	for _, td := range taskDurations {
		for _, sinks := range sinkCounts {
			for _, outcome := range []struct {
				err     bool
				newTask func(d time.Duration) func(context.Context, benchPayload) error
				wantErr error
			}{
				{err: false, newTask: poolSinkTask, wantErr: nil},
				{err: true, newTask: poolFailingSinkTask, wantErr: ErrWorkerFailed},
			} {
				var errTitle string
				if outcome.err {
					errTitle = "/witherr"
				}

				b.Run(fmt.Sprintf("sinks=%d/task_duration=%s%s", sinks, td.name, errTitle),
					func(b *testing.B) {
						ctx := context.Background()
						task := outcome.newTask(td.d)
						payload := benchPayload{
							Version:   "1",
							SinkType:  "kafka",
							SinkName:  "orders",
							Timestamp: time.Unix(0, 0),
							Operation: "INSERT",
							Database:  "hump",
							Schema:    "public",
							Table:     "orders",
							Headers:   map[string]string{"source": "wal"},
							Prev:      nil,
							Next:      map[string]any{"id": 1},
						}

						wp := New[benchPayload](
							WithWorkers(sinks),
							WithQueueSize(sinks),
							WithErrorBufferSize(1024),
						)

						drained := make(chan struct{})
						go func() {
							defer close(drained)
							for range wp.Errors() {
							}
						}()

						b.ReportAllocs()
						for b.Loop() {
							if err := wp.BeginBatch(); err != nil {
								b.Fatal(err)
							}

							for range sinks {
								if err := wp.Submit(ctx, task, payload); err != nil {
									_ = wp.FlushBatch()
									b.Fatal(err)
								}
							}

							if err := wp.FlushBatch(); !errors.Is(err, outcome.wantErr) {
								b.Fatalf("FlushBatch: got %v, want %v", err, outcome.wantErr)
							}
						}

						b.ReportMetric(float64(b.N)*float64(sinks)/b.Elapsed().Seconds(), "tasks/s")

						_ = wp.Shutdown()
						<-drained
					})
			}
		}
	}
}

func poolSinkTask(d time.Duration) func(context.Context, benchPayload) error {
	if d == 0 {
		return func(context.Context, benchPayload) error {
			return nil
		}
	}

	return func(context.Context, benchPayload) error {
		time.Sleep(d)
		return nil
	}
}

func poolFailingSinkTask(d time.Duration) func(context.Context, benchPayload) error {
	if d == 0 {
		return func(context.Context, benchPayload) error {
			return errTask
		}
	}

	return func(context.Context, benchPayload) error {
		time.Sleep(d)
		return errTask
	}
}

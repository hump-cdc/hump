package workerpool

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func BenchmarkBaselineWaitGroup(b *testing.B) {
	for _, td := range taskDurations {
		for _, sinks := range sinkCounts {
			b.Run(fmt.Sprintf("sinks=%d/task_duration=%s", sinks, td.name),
				func(b *testing.B) {
					ctx := context.Background()
					task := baselineSinkTask(td.d)
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

					b.ReportAllocs()
					for b.Loop() {
						var wg sync.WaitGroup

						wg.Add(sinks)
						for range sinks {
							go func() {
								defer wg.Done()
								_ = task(ctx, payload)
							}()
						}

						wg.Wait()
					}

					b.ReportMetric(float64(b.N)*float64(sinks)/b.Elapsed().Seconds(), "tasks/s")
				})
		}
	}
}

func BenchmarkBaselineErrgroup(b *testing.B) {
	for _, td := range taskDurations {
		for _, sinks := range sinkCounts {
			b.Run(fmt.Sprintf("sinks=%d/task_duration=%s", sinks, td.name),
				func(b *testing.B) {
					ctx := context.Background()
					task := baselineSinkTask(td.d)
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

					b.ReportAllocs()
					for b.Loop() {
						var g errgroup.Group

						// The limit matches the pool's worker count, the way the
						// queue in the pool matches its batch: it never binds here.
						g.SetLimit(sinks)
						for range sinks {
							g.Go(func() error { return task(ctx, payload) })
						}

						if err := g.Wait(); err != nil {
							b.Fatal(err)
						}
					}

					b.ReportMetric(float64(b.N)*float64(sinks)/b.Elapsed().Seconds(), "tasks/s")
				})
		}
	}
}

func BenchmarkBaselineSemaphore(b *testing.B) {
	for _, td := range taskDurations {
		for _, sinks := range sinkCounts {
			b.Run(fmt.Sprintf("sinks=%d/task_duration=%s", sinks, td.name),
				func(b *testing.B) {
					ctx := context.Background()
					task := baselineSinkTask(td.d)
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

					// The channel is allocated once, like the pool's queue, so the
					// per-batch cost is the goroutines and the token round trip.
					sem := make(chan struct{}, sinks)

					b.ReportAllocs()
					for b.Loop() {
						var wg sync.WaitGroup

						wg.Add(sinks)
						for range sinks {
							sem <- struct{}{}
							go func() {
								defer wg.Done()
								defer func() { <-sem }()
								_ = task(ctx, payload)
							}()
						}

						wg.Wait()
					}

					b.ReportMetric(float64(b.N)*float64(sinks)/b.Elapsed().Seconds(), "tasks/s")
				})
		}
	}
}

func baselineSinkTask(d time.Duration) func(context.Context, benchPayload) error {
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

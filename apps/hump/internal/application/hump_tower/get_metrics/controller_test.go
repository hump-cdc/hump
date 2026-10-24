package getmetrics

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"hump/internal/domain"
	metricscollector "hump/internal/port/metrics_collector"

	"github.com/danielgtaylor/huma/v2"
)

func TestHandlerRejectsDisabledTelemetry(t *testing.T) {
	_, err := call(NewController(NewUseCase(nil)), "1m")

	if status(err) != 503 {
		t.Fatalf("error = %v, want 503", err)
	}
}

func TestHandlerRejectsAnUnknownDuration(t *testing.T) {
	_, err := call(NewController(NewUseCase(&fakeCollector{})), "7m")

	if status(err) != 422 {
		t.Fatalf("error = %v, want 422", err)
	}
}

func TestHandlerRendersTheSnapshot(t *testing.T) {
	p95 := 12.5
	success := 0.75
	ts := time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)
	table := domain.TableIdentifier{Schema: "public", Name: "users"}

	collector := &fakeCollector{snapshot: metricscollector.Snapshot{
		Points: []metricscollector.Point{{
			Timestamp:           ts,
			Delivered:           2.6,
			Failed:              1.4,
			Throughput:          0.5,
			ReplicationLagBytes: 1024,
			MemoryAllocBytes:    2048,
			SinkDelivery:        map[string]float64{"kafka": 3.5},
			SinkLatencyP95:      map[string]float64{"kafka": 7},
		}},
		Tables: map[string]metricscollector.TableOps{"public.users": {
			Insert:   metricscollector.TableOp{Count: 1, P95: &p95},
			Update:   metricscollector.TableOp{Count: 2},
			Delete:   metricscollector.TableOp{Count: 3},
			Truncate: metricscollector.TableOp{Count: 4},
		}},
		Sinks: []metricscollector.Sink{{
			Type: "kafka",
			Host: "broker",
			Targets: []metricscollector.Target{{
				Target:   "users",
				Delivery: metricscollector.Delivery{Operations: 4, Success: &success, P95: &p95},
			}},
			Delivery: metricscollector.Delivery{Operations: 4, Success: &success, P95: &p95},
		}},
		Connections: []metricscollector.ConnectionOps{{
			Connection: metricscollector.Connection{Table: table, SinkType: "kafka", Host: "broker", Target: "users"},
			Success:    3,
			Failure:    1,
		}},
	}}

	resp, err := call(NewController(NewUseCase(collector)), "5m")
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	body := resp.Body
	if body.Duration != "5m" || body.BucketSeconds != 5*60 {
		t.Errorf("Duration, BucketSeconds = %q, %d, want 5m, 300", body.Duration, body.BucketSeconds)
	}

	wantPoints := []MetricsPoint{{
		Timestamp:           ts,
		Delivered:           3,
		Failed:              1,
		Throughput:          0.5,
		ReplicationLagBytes: 1024,
		MemoryAllocBytes:    2048,
		SinkDelivery:        map[string]int64{"kafka": 4},
		SinkLatencyP95:      map[string]float64{"kafka": 7},
	}}
	if !reflect.DeepEqual(body.Points, wantPoints) {
		t.Errorf("Points = %+v, want %+v", body.Points, wantPoints)
	}

	wantTables := map[string]MetricsTableOps{"public.users": {
		Insert:   MetricsTableOp{Count: 1, P95: &p95},
		Update:   MetricsTableOp{Count: 2},
		Delete:   MetricsTableOp{Count: 3},
		Truncate: MetricsTableOp{Count: 4},
	}}
	if !reflect.DeepEqual(body.Tables, wantTables) {
		t.Errorf("Tables = %+v, want %+v", body.Tables, wantTables)
	}

	wantSinks := []MetricsSink{{
		Type:       "kafka",
		Host:       "broker",
		Operations: 4,
		Success:    &success,
		P95:        &p95,
		Targets:    []MetricsSinkTarget{{Target: "users", Operations: 4, Success: &success, P95: &p95}},
	}}
	if !reflect.DeepEqual(body.Sinks, wantSinks) {
		t.Errorf("Sinks = %+v, want %+v", body.Sinks, wantSinks)
	}

	wantConnections := []MetricsConnection{{
		Table: "public.users", Sink: "kafka", Host: "broker", Target: "users", Success: 3, Failure: 1,
	}}
	if !reflect.DeepEqual(body.Connections, wantConnections) {
		t.Errorf("Connections = %+v, want %+v", body.Connections, wantConnections)
	}
}

// call runs the handler for a request of duration.
func call(c *Controller, duration string) (*MetricsResponse, error) {
	input := &MetricsRequest{}
	input.Body.Duration = duration

	return c.Handler()(context.Background(), input)
}

// status returns the HTTP status huma put on err, or 0 when it carries none.
func status(err error) int {
	var statusErr huma.StatusError
	if errors.As(err, &statusErr) {
		return statusErr.GetStatus()
	}

	return 0
}

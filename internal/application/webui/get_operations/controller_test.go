package getoperations

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"hump/internal/domain"
	operationscollector "hump/internal/port/operations_collector"

	"github.com/danielgtaylor/huma/v2"
)

func TestHandlerRejectsADisabledLog(t *testing.T) {
	_, err := call(NewController(NewUseCase(nil)), request{})

	if status(err) != 503 {
		t.Fatalf("error = %v, want 503", err)
	}
}

func TestHandlerRejectsAnInvertedWindow(t *testing.T) {
	_, err := call(NewController(NewUseCase(&fakeCollector{})), request{from: 50, to: 10})

	if status(err) != 422 {
		t.Fatalf("error = %v, want 422", err)
	}
}

func TestHandlerPassesTheFilterToTheCollector(t *testing.T) {
	collector := &fakeCollector{}

	_, err := call(NewController(NewUseCase(collector)), request{
		from:       10,
		to:         35,
		sinks:      []string{"kafka", "nats"},
		hosts:      []string{"broker-1"},
		targets:    []string{"users"},
		tables:     []string{"public.users"},
		operations: []string{"insert", "delete"},
		statuses:   []string{"failure", "success"},
		text:       "boom",
	})
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	want := operationscollector.Query{
		SinkTypes:  []string{"kafka", "nats"},
		Hosts:      []string{"broker-1"},
		Targets:    []string{"users"},
		Tables:     []string{"public.users"},
		Operations: []domain.OperationType{domain.OperationTypeInsert, domain.OperationTypeDelete},
		Statuses:   []domain.DeliveryStatus{domain.DeliveryStatusFailure, domain.DeliveryStatusSuccess},
		Text:       "boom",
		Start:      10,
		End:        35,
	}

	if got := collector.query; !queriesEqual(got, want) {
		t.Fatalf("query = %+v, want %+v", got, want)
	}
}

func TestHandlerReportsTheClampedWindow(t *testing.T) {
	collector := &fakeCollector{page: operationscollector.Page{Total: 3, Start: 0, End: 3}}

	resp, err := call(NewController(NewUseCase(collector)), request{to: 50})
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	if resp.Body.From != 0 || resp.Body.To != 3 {
		t.Fatalf("from, to = %d, %d, want the window clamped to the 3 matches", resp.Body.From, resp.Body.To)
	}
}

func TestHandlerRendersARecord(t *testing.T) {
	started := time.Now()
	collector := &fakeCollector{page: operationscollector.Page{
		Total:   7,
		Written: 42,
		Records: []operationscollector.Record{{
			Seq:         3,
			StartedAt:   started,
			Duration:    1500 * time.Microsecond,
			Table:       domain.TableIdentifier{Schema: "public", Name: "users"},
			Operation:   domain.OperationTypeInsert,
			SinkType:    "kafka",
			Status:      domain.DeliveryStatusFailure,
			Error:       "boom",
			Retries:     1,
			Payload:     []byte(`{"id":1}`),
			PayloadSize: 8,
			Attempts: []operationscollector.Attempt{
				{Duration: 2 * time.Millisecond, Status: domain.DeliveryStatusRetry, Error: "boom"},
				{Duration: 3 * time.Millisecond, Status: domain.DeliveryStatusFailure, Error: "boom"},
			},
		}},
	}}

	resp, err := call(NewController(NewUseCase(collector)), request{})
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	if resp.Body.Total != 7 || resp.Body.Written != 42 || len(resp.Body.Items) != 1 {
		t.Fatalf("body = %+v, want the page's totals and one item", resp.Body)
	}

	item := resp.Body.Items[0]
	if item.Table != "public.users" {
		t.Fatalf("Table = %q, want the schema-qualified name", item.Table)
	}

	if item.Payload != `{"id":1}` {
		t.Fatalf("Payload = %q, want the body as text rather than base64", item.Payload)
	}

	// 1.5ms rounds to 2ms, which is what every latency on the API is carried in.
	if item.Duration != 2 {
		t.Fatalf("Duration = %d, want 2 milliseconds", item.Duration)
	}

	if len(item.Attempts) != 2 || item.Attempts[0].Attempt != 1 || item.Attempts[1].Attempt != 2 {
		t.Fatalf("attempts = %+v, want them numbered from 1", item.Attempts)
	}
}

// request is the body of one call, flattened so a test names only what it
// cares about.
type request struct {
	from       int
	to         int
	sinks      []string
	hosts      []string
	targets    []string
	tables     []string
	operations []string
	statuses   []string
	text       string
}

// call runs the controller's handler over req.
func call(c *Controller, req request) (*OperationsResponse, error) {
	input := &OperationsRequest{}
	input.Body.From = req.from
	input.Body.To = req.to
	input.Body.Sinks = req.sinks
	input.Body.Hosts = req.hosts
	input.Body.Targets = req.targets
	input.Body.Tables = req.tables
	input.Body.Operations = req.operations
	input.Body.Statuses = req.statuses
	input.Body.Text = req.text

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

// queriesEqual compares two queries field by field, since a Query holding
// slices cannot be compared with ==.
func queriesEqual(a, b operationscollector.Query) bool {
	return slices.Equal(a.SinkTypes, b.SinkTypes) && slices.Equal(a.Hosts, b.Hosts) &&
		slices.Equal(a.Targets, b.Targets) && slices.Equal(a.Tables, b.Tables) &&
		slices.Equal(a.Operations, b.Operations) && slices.Equal(a.Statuses, b.Statuses) &&
		a.Text == b.Text && a.Start == b.Start && a.End == b.End
}

// fakeCollector answers every search with the same page and keeps the query it
// was asked for. Only Search is reachable from the controller.
type fakeCollector struct {
	query operationscollector.Query
	page  operationscollector.Page
}

func (c *fakeCollector) Connect()                               {}
func (c *fakeCollector) Disconnect()                            {}
func (c *fakeCollector) Connected() bool                        { return true }
func (c *fakeCollector) Record(*operationscollector.Record)     {}
func (c *fakeCollector) Dump(int, int) operationscollector.Page { return c.page }

func (c *fakeCollector) Search(q operationscollector.Query) operationscollector.Page {
	c.query = q

	return c.page
}

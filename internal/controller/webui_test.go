package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	operationscollectoradapter "hump/internal/adapter/operations_collector"
	getwebuiconfig "hump/internal/application/webui/get_webui_config"
	getwebuimetrics "hump/internal/application/webui/get_webui_metrics"
	getwebuioperations "hump/internal/application/webui/get_webui_operations"
	streamwebuiupdates "hump/internal/application/webui/stream_webui_updates"
	"hump/internal/domain"
	operationscollectorport "hump/internal/port/operations_collector"
	"hump/pkg/app/options"

	"hump/config"

	"github.com/go-chi/chi/v5"
)

func TestOperationsRouteAnswersTheMatchingRecords(t *testing.T) {
	collector := newTestOperationsCollector(t)
	router := newTestRouter(collector)

	record(collector, "users", domain.DeliveryStatusSuccess)
	record(collector, "orders", domain.DeliveryStatusFailure)

	body := post(t, router, `{"from":0,"to":50,"status":"failure"}`)

	if body.Total != 1 || len(body.Items) != 1 {
		t.Fatalf("body = %+v, want one failed delivery", body)
	}

	item := body.Items[0]
	if item.Table != "public.orders" || item.Status != "failure" || item.Error != "boom" {
		t.Fatalf("item = %+v, want the failed delivery of public.orders", item)
	}

	// A payload goes out as text rather than the base64 a []byte would produce,
	// and a duration as whole milliseconds.
	if item.Payload != `{"id":1}` || item.Duration != 2 {
		t.Fatalf("payload, duration = %q, %d, want the body as text and 2ms", item.Payload, item.Duration)
	}

	if len(item.Attempts) != 1 || item.Attempts[0].Attempt != 1 {
		t.Fatalf("attempts = %+v, want one numbered from 1", item.Attempts)
	}

	// A sink that never filled the capture leaves the map nil, and the client's
	// schema has no null to parse.
	if item.Metadata == nil {
		t.Fatal("metadata = null, want an empty object")
	}
}

func TestOperationsRouteAnswersNewestFirst(t *testing.T) {
	collector := newTestOperationsCollector(t)
	router := newTestRouter(collector)

	for _, table := range []string{"first", "second", "third"} {
		record(collector, table, domain.DeliveryStatusSuccess)
	}

	body := post(t, router, `{"from":0,"to":50}`)

	want := []string{"public.third", "public.second", "public.first"}
	if len(body.Items) != len(want) {
		t.Fatalf("got %d items, want %d", len(body.Items), len(want))
	}

	for i, table := range want {
		if body.Items[i].Table != table {
			t.Fatalf("item %d = %q, want %q: the newest delivery comes first", i, body.Items[i].Table, table)
		}
	}
}

func TestOperationsRouteCutsTheWindowByIndex(t *testing.T) {
	collector := newTestOperationsCollector(t)
	router := newTestRouter(collector)

	for _, table := range []string{"first", "second", "third", "fourth"} {
		record(collector, table, domain.DeliveryStatusSuccess)
	}

	body := post(t, router, `{"from":1,"to":3}`)

	// from and to index the result set, not time: 1 to 3 is the second and
	// third newest.
	want := []string{"public.third", "public.second"}
	if len(body.Items) != len(want) {
		t.Fatalf("got %d items, want %d", len(body.Items), len(want))
	}

	for i, table := range want {
		if body.Items[i].Table != table {
			t.Fatalf("item %d = %q, want %q", i, body.Items[i].Table, table)
		}
	}

	// Total stays the whole match count, so a client knows how far it can page.
	if body.Total != 4 || body.From != 1 || body.To != 3 {
		t.Fatalf("total, from, to = %d, %d, %d, want 4, 1, 3", body.Total, body.From, body.To)
	}
}

func TestOperationsRouteClampsAWindowPastTheEnd(t *testing.T) {
	collector := newTestOperationsCollector(t)
	router := newTestRouter(collector)
	record(collector, "users", domain.DeliveryStatusSuccess)

	body := post(t, router, `{"from":0,"to":5000}`)

	if len(body.Items) != 1 || body.From != 0 || body.To != 1 {
		t.Fatalf("items, from, to = %d, %d, %d, want the window clamped to the one record", len(body.Items), body.From, body.To)
	}
}

func TestOperationsRouteRejectsAWindowlessBody(t *testing.T) {
	router := newTestRouter(newTestOperationsCollector(t))

	resp := request(router, `{"status":"failure"}`)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 for a body without a window", resp.Code)
	}
}

// newTestOperationsCollector returns a connected collector that is closed when
// the test ends.
func newTestOperationsCollector(t *testing.T) *operationscollectoradapter.Collector {
	t.Helper()

	c := operationscollectoradapter.New()
	c.Connect()
	t.Cleanup(c.Disconnect)

	return c
}

// newTestRouter returns a router carrying the web UI API, with only the
// operations endpoint backed by anything.
func newTestRouter(collector operationscollectorport.Collector) chi.Router {
	router := chi.NewMux()
	NewWebUIController(
		getwebuiconfig.NewController(options.Options{}, config.Configuration{}, nil),
		getwebuimetrics.NewController(nil),
		getwebuioperations.NewController(collector),
		streamwebuiupdates.NewController(nil, nil, nil),
	).Api(router)

	return router
}

// record stores one delivery of table, with the payload and attempt a sink
// would have filled in.
func record(collector operationscollectorport.Collector, table string, status domain.DeliveryStatus) {
	rec := operationscollectorport.Record{
		StartedAt: time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC),
		Duration:  2 * time.Millisecond,
		Table:     domain.TableIdentifier{Schema: "public", Name: domain.TableName(table)},
		Operation: domain.OperationTypeInsert,
		SinkType:  "kafka",
		Target:    table,
		Status:    status,
		Payload:   []byte(`{"id":1}`),
		Attempts:  []operationscollectorport.Attempt{{Status: status}},
	}

	if status == domain.DeliveryStatusFailure {
		rec.Error = "boom"
	}

	collector.Record(&rec)
}

// operationsBody is the response as a client sees it, which is how the test
// catches a field that marshals into something unusable.
type operationsBody struct {
	Total int `json:"total"`
	From  int `json:"from"`
	To    int `json:"to"`
	Items []struct {
		Table    string            `json:"table"`
		Status   string            `json:"status"`
		Error    string            `json:"error"`
		Payload  string            `json:"payload"`
		Duration int64             `json:"duration"`
		Metadata map[string]string `json:"metadata"`
		Attempts []struct {
			Attempt int `json:"attempt"`
		} `json:"attempts"`
	} `json:"items"`
}

// post issues one operations request and decodes the body it answers with.
func post(t *testing.T, router chi.Router, body string) operationsBody {
	t.Helper()

	resp := request(router, body)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body)
	}

	var decoded operationsBody
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return decoded
}

// request posts body to the operations endpoint.
func request(router chi.Router, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/operations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	return resp
}

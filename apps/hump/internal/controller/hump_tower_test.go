package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	operationscollectoradapter "hump/internal/adapter/operations_collector"
	getconfig "hump/internal/application/hump_tower/get_config"
	getmetrics "hump/internal/application/hump_tower/get_metrics"
	getoperations "hump/internal/application/hump_tower/get_operations"
	getsnapshotestimates "hump/internal/application/hump_tower/get_snapshot_estimates"
	getsnapshottables "hump/internal/application/hump_tower/get_snapshot_tables"
	startsnapshot "hump/internal/application/hump_tower/start_snapshot"
	stopsnapshot "hump/internal/application/hump_tower/stop_snapshot"
	streamsnapshot "hump/internal/application/hump_tower/stream_snapshot"
	streamupdates "hump/internal/application/hump_tower/stream_updates"
	"hump/internal/domain"
	"hump/internal/port/database"
	operationscollectorport "hump/internal/port/operations_collector"
	"hump/internal/port/snapshot"
	"hump/pkg/app/options"
	"hump/pkg/middleware"

	"hump/config"

	"github.com/go-chi/chi/v5"
)

func TestOperationsRouteAnswersTheMatchingRecords(t *testing.T) {
	collector := newTestOperationsCollector(t)
	router := newTestRouter(collector)

	record(collector, "users", domain.DeliveryStatusSuccess)
	record(collector, "orders", domain.DeliveryStatusFailure)

	body := post(t, router, `{"from":0,"to":50,"statuses":["failure"]}`)

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

	resp := request(router, `{"statuses":["failure"]}`)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 for a body without a window", resp.Code)
	}
}

func TestSnapshotEstimateRouteStreamsEvents(t *testing.T) {
	router := newTestRouter(newTestOperationsCollector(t))

	req := httptest.NewRequest(http.MethodPost, "/api/snapshot/estimate", strings.NewReader(`{"tables":[{"schema":"public","table":"users","where":"id > 1"}]}`))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK || !strings.HasPrefix(resp.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status, content type = %d, %q, want 200 and an event stream", resp.Code, resp.Header().Get("Content-Type"))
	}

	if !strings.Contains(resp.Body.String(), "event: done") {
		t.Fatalf("body = %q, want a done event", resp.Body)
	}
}

func TestSnapshotEstimateRouteRejectsATablelessBody(t *testing.T) {
	router := newTestRouter(newTestOperationsCollector(t))

	req := httptest.NewRequest(http.MethodPost, "/api/snapshot/estimate", strings.NewReader(`{"tables":[{"where":"id > 1"}]}`))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 for a table without a name", resp.Code)
	}
}

func TestSnapshotRoutesRejectUnconfiguredTargets(t *testing.T) {
	cfg := config.Configuration{Tables: []config.Table{{Schema: "public", Name: "users"}}}
	db := &snapshotTestDatabase{}
	runner := &snapshotTestRunner{}
	router := newTestRouterWithSnapshot(nil, cfg, db, runner)
	body := `{"tables":[{"table":"users"},{"schema":"pg_catalog","table":"pg_authid"}]}`
	for _, path := range []string{"/api/snapshot/start", "/api/snapshot/estimate"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if path == "/api/snapshot/start" {
			if resp.Code != http.StatusUnprocessableEntity || runner.calls != 0 {
				t.Fatalf("start status = %d, runner calls = %d; want 422 and zero calls", resp.Code, runner.calls)
			}
			continue
		}
		if resp.Code != http.StatusOK || !strings.HasPrefix(resp.Header().Get("Content-Type"), "text/event-stream") {
			t.Fatalf("estimate response = %d, %s", resp.Code, resp.Body)
		}
		for _, fragment := range []string{"event: estimate", `"schema":""`, `"count":3`, "event: error", `"schema":"pg_catalog"`, `"table":"pg_authid"`, "event: done"} {
			if !strings.Contains(resp.Body.String(), fragment) {
				t.Errorf("estimate body %q missing %q", resp.Body, fragment)
			}
		}
	}
	if len(db.calls) != 1 || db.calls[0] != (domain.TableIdentifier{Schema: "public", Name: "users"}) {
		t.Fatalf("database calls = %+v, want only public.users", db.calls)
	}
}

func TestSnapshotMutationRoutesEnforceBrowserPolicy(t *testing.T) {
	for _, path := range []string{"/api/snapshot/start", "/api/snapshot/stop"} {
		for _, tt := range []struct {
			name        string
			contentType string
			origins     []string
			cors        string
			want        int
		}{
			{"missing content type", "", nil, "", http.StatusUnsupportedMediaType},
			{"simple form", "application/x-www-form-urlencoded", nil, "", http.StatusUnsupportedMediaType},
			{"malformed content type", "application/json; charset", nil, "", http.StatusUnsupportedMediaType},
			{"media type before origin", "text/plain", []string{"null"}, "", http.StatusUnsupportedMediaType},
			{"cross origin", "application/json", []string{"https://evil.test"}, "", http.StatusForbidden},
			{"empty origin", "application/json", []string{""}, "", http.StatusForbidden},
			{"null origin", "application/json", []string{"null"}, "null,*", http.StatusForbidden},
			{"multiple origins", "application/json", []string{"http://example.com", "http://example.com"}, "", http.StatusForbidden},
			{"malformed origin", "application/json", []string{"https://ui.test/"}, "https://ui.test/", http.StatusForbidden},
			{"wildcard", "application/json", []string{"https://evil.test"}, "*", http.StatusForbidden},
			{"CLI", "application/json", nil, "", 0},
			{"same origin", "application/json", []string{"http://example.com"}, "", 0},
			{"JSON parameters", "application/json; charset=utf-8", []string{"http://example.com"}, "", 0},
			{"explicit CORS", "application/json", []string{"https://ui.test"}, " https://ui.test ,", 0},
		} {
			t.Run(path+"/"+tt.name, func(t *testing.T) {
				t.Setenv("CORS", tt.cors)
				runner := &snapshotTestRunner{}
				cfg := config.Configuration{Tables: []config.Table{{Schema: "public", Name: "users"}}}
				router := newTestRouterWithSnapshot(nil, cfg, nil, runner)
				body := ""
				if path == "/api/snapshot/start" {
					body = `{"tables":[{"schema":"public","table":"users"}]}`
				}
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				if tt.contentType != "" {
					req.Header.Set("Content-Type", tt.contentType)
				}
				if tt.origins != nil {
					req.Header["Origin"] = tt.origins
				}
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				want := tt.want
				wantStarts, wantStops := 0, 0
				if want == 0 {
					if path == "/api/snapshot/start" {
						want, wantStarts = http.StatusAccepted, 1
					} else {
						want, wantStops = http.StatusNoContent, 1
					}
				}
				if resp.Code != want || runner.calls != wantStarts || runner.stopCalls != wantStops {
					t.Fatalf("status, start calls, stop calls = %d, %d, %d; want %d, %d, %d; body = %s",
						resp.Code, runner.calls, runner.stopCalls, want, wantStarts, wantStops, resp.Body)
				}
			})
		}
	}
}

func TestEmbeddedShellPolicyAllowsTheThemeScript(t *testing.T) {
	handler := middleware.Static(humpTowerFS, "hump_tower")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	if strings.Count(csp, "'sha256-") != 1 {
		t.Fatalf("Content-Security-Policy = %q, want exactly one script hash: the inline theme script", csp)
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
	return newTestRouterWithSnapshot(collector, config.Configuration{}, nil, nil)
}

// newTestRouterWithSnapshot injects configured snapshot dependencies into the API.
func newTestRouterWithSnapshot(collector operationscollectorport.Collector, cfg config.Configuration, db database.Database, runner snapshot.Runner) chi.Router {
	router := chi.NewMux()
	NewHumpTowerController(
		getconfig.NewController(getconfig.NewUseCase(options.Options{}, config.Configuration{}, nil)),
		getmetrics.NewController(getmetrics.NewUseCase(nil)),
		getoperations.NewController(getoperations.NewUseCase(collector)),
		streamupdates.NewController(streamupdates.NewUseCase(nil, nil, nil, nil)),
		getsnapshotestimates.NewController(getsnapshotestimates.NewUseCase(cfg, db)),
		getsnapshottables.NewController(getsnapshottables.NewUseCase(config.Configuration{}, nil)),
		streamsnapshot.NewController(streamsnapshot.NewUseCase(nil)),
		startsnapshot.NewController(startsnapshot.NewUseCase(cfg, runner)),
		stopsnapshot.NewController(stopsnapshot.NewUseCase(runner)),
	).Api(router)

	return router
}

// snapshotTestDatabase records count targets and returns one count.
type snapshotTestDatabase struct {
	database.Database
	calls []domain.TableIdentifier
}

// GetRowCount records the resolved target.
func (d *snapshotTestDatabase) GetRowCount(_ context.Context, table domain.TableIdentifier, _ string) (<-chan database.RowCount, error) {
	d.calls = append(d.calls, table)
	counts := make(chan database.RowCount, 1)
	counts <- database.RowCount{Num: 3}
	close(counts)
	return counts, nil
}

// snapshotTestRunner records snapshot mutations without doing any work.
type snapshotTestRunner struct {
	snapshot.Runner
	calls     int
	stopCalls int
}

// Start records a call without doing any work.
func (r *snapshotTestRunner) Start(_ []snapshot.Request) error {
	r.calls++
	return nil
}

// Stop records a call without doing any work.
func (r *snapshotTestRunner) Stop() error {
	r.stopCalls++
	return nil
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

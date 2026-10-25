package metric

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPrefixesFactoryMetrics(t *testing.T) {
	factory, handler, gatherer := New("hump")

	factory.NewCounter(prometheus.CounterOpts{Name: "events_total", Help: "Events."}).Inc()

	families, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, want := range []string{"hump_events_total", "go_goroutines"} {
		if !names[want] {
			t.Errorf("metric %q not gathered", want)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "hump_events_total 1") {
		t.Errorf("handler output lacks hump_events_total 1:\n%s", body)
	}
}

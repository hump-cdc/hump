package controller

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	getconfig "hump/internal/application/hump_tower/get_config"
	getmetrics "hump/internal/application/hump_tower/get_metrics"
	getoperations "hump/internal/application/hump_tower/get_operations"
	getsnapshotestimates "hump/internal/application/hump_tower/get_snapshot_estimates"
	getsnapshottables "hump/internal/application/hump_tower/get_snapshot_tables"
	startsnapshot "hump/internal/application/hump_tower/start_snapshot"
	stopsnapshot "hump/internal/application/hump_tower/stop_snapshot"
	streamsnapshot "hump/internal/application/hump_tower/stream_snapshot"
	streamupdates "hump/internal/application/hump_tower/stream_updates"
	"hump/internal/port/database"
	"hump/pkg/app/options"
	"hump/pkg/app/shutdown"

	"hump/config"
)

func TestServeFailsOnABusyPort(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	port := uint(listener.Addr().(*net.TCPAddr).Port)

	if err := NewObservabilityController(okHandler("health"), okHandler("metrics")).Serve(port); err == nil {
		t.Error("observability Serve() = nil, want the listen error")
	}

	if err := newTestHumpTowerController().Serve(context.Background(), port); err == nil {
		t.Error("web UI Serve() = nil, want the listen error")
	}
}

// The shutdown registry is process-global and runs once, so both servers are
// served and stopped by one test.
func TestServeAnswersUntilShutdown(t *testing.T) {
	observabilityPort, webPort := freePort(t), freePort(t)

	observabilityDone := make(chan error, 1)
	go func() {
		observabilityDone <- NewObservabilityController(okHandler("health"), okHandler("metrics")).Serve(observabilityPort)
	}()

	webDone := make(chan error, 1)
	go func() {
		webDone <- newTestHumpTowerController().Serve(context.Background(), webPort)
	}()

	for path, want := range map[string]string{
		"/health/startup": "health",
		"/health/ready":   "health",
		"/health/live":    "health",
		"/metrics":        "metrics",
	} {
		if body := awaitGet(t, observabilityPort, path); body != want {
			t.Errorf("GET %s = %q, want %q", path, body, want)
		}
	}

	if body := awaitGet(t, webPort, "/api/config"); !strings.Contains(body, `"tables"`) {
		t.Errorf("GET /api/config = %q, want the config description", body)
	}

	ran := make(chan struct{})
	shutdown.OnShutdown(shutdown.StageWorkers, "controller test probe", func(context.Context) error {
		close(ran)
		return nil
	})

	_ = shutdown.Shutdown(5 * time.Second)

	select {
	case <-ran:
	default:
		t.Skip("the global shutdown already ran in this process, so the servers cannot be stopped")
	}

	for name, done := range map[string]chan error{"observability": observabilityDone, "web UI": webDone} {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("%s Serve() = %v, want nil on a clean shutdown", name, err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("%s Serve() did not return after shutdown", name)
		}
	}
}

// okHandler answers every request with body.
func okHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
}

// newTestHumpTowerController returns a web UI controller over nil dependencies.
func newTestHumpTowerController() *HumpTowerController {
	return NewHumpTowerController(
		getconfig.NewController(getconfig.NewUseCase(options.Options{}, config.Configuration{}, fakeDatabase{})),
		getmetrics.NewController(getmetrics.NewUseCase(nil)),
		getoperations.NewController(getoperations.NewUseCase(nil)),
		streamupdates.NewController(streamupdates.NewUseCase(nil, nil, nil, nil)),
		getsnapshotestimates.NewController(getsnapshotestimates.NewUseCase(nil)),
		getsnapshottables.NewController(getsnapshottables.NewUseCase(config.Configuration{}, nil)),
		streamsnapshot.NewController(streamsnapshot.NewUseCase(nil)),
		startsnapshot.NewController(startsnapshot.NewUseCase(nil)),
		stopsnapshot.NewController(stopsnapshot.NewUseCase(nil)),
	)
}

// freePort returns a TCP port that was free a moment ago.
func freePort(t *testing.T) uint {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	return uint(listener.Addr().(*net.TCPAddr).Port)
}

// awaitGet polls GET path on the local port until the server answers 200, and
// returns the body.
func awaitGet(t *testing.T, port uint, path string) string {
	t.Helper()

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	deadline := time.Now().Add(5 * time.Second)

	for {
		resp, err := http.Get(url) //nolint:noctx // a test against a local server
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if readErr != nil || resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s: status %d, read error %v", path, resp.StatusCode, readErr)
			}

			return string(body)
		}

		if time.Now().After(deadline) {
			t.Fatalf("GET %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// fakeDatabase answers the getters the config description reads.
type fakeDatabase struct {
	database.Database
}

func (fakeDatabase) GetDbHost() string          { return "db.local" }
func (fakeDatabase) GetDbPort() uint16          { return 5432 }
func (fakeDatabase) GetDbName() string          { return "app" }
func (fakeDatabase) GetPublicationName() string { return "hump_pub" }
func (fakeDatabase) GetSlotName() string        { return "hump_slot" }

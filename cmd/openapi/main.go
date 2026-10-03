package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"hump/config"
	getconfig "hump/internal/application/webui/get_config"
	getmetrics "hump/internal/application/webui/get_metrics"
	getoperations "hump/internal/application/webui/get_operations"
	getsnapshotestimates "hump/internal/application/webui/get_snapshot_estimates"
	getsnapshottables "hump/internal/application/webui/get_snapshot_tables"
	startsnapshot "hump/internal/application/webui/start_snapshot"
	stopsnapshot "hump/internal/application/webui/stop_snapshot"
	streamsnapshot "hump/internal/application/webui/stream_snapshot"
	streamupdates "hump/internal/application/webui/stream_updates"
	"hump/internal/controller"
	"hump/pkg/app/options"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

const outEnv = "OPENAPI_OUT"

// sseMediaType is the content type huma documents an SSE response under.
const sseMediaType = "text/event-stream"

func main() {
	// Handlers are only registered, never invoked, so their dependencies can stay empty.
	api := controller.NewWebUIController(
		getconfig.NewController(getconfig.NewUseCase(options.Options{}, config.Configuration{}, nil)),
		getmetrics.NewController(getmetrics.NewUseCase(nil)),
		getoperations.NewController(getoperations.NewUseCase(nil)),
		streamupdates.NewController(streamupdates.NewUseCase(nil, nil, nil)),
		getsnapshotestimates.NewController(getsnapshotestimates.NewUseCase(nil)),
		getsnapshottables.NewController(getsnapshottables.NewUseCase(config.Configuration{}, nil)),
		streamsnapshot.NewController(streamsnapshot.NewUseCase(nil)),
		startsnapshot.NewController(startsnapshot.NewUseCase(nil)),
		stopsnapshot.NewController(stopsnapshot.NewUseCase(nil)),
	).Api(chi.NewMux())

	if err := flattenSSESchema(
		api.OpenAPI(),
		http.MethodGet,
		"/api/live",
		"#/components/schemas/OperationsEvent",
		"#/components/schemas/WelcomeEvent",
	); err != nil {
		log.Fatalf("flatten sse schema: %v", err)
	}

	if err := flattenSSESchema(
		api.OpenAPI(),
		http.MethodPost,
		"/api/snapshot/estimate",
		"#/components/schemas/EstimateEvent",
		"#/components/schemas/EstimateErrorEvent",
		"#/components/schemas/EstimatesDoneEvent",
	); err != nil {
		log.Fatalf("flatten sse schema: %v", err)
	}

	if err := flattenSSESchema(
		api.OpenAPI(),
		http.MethodGet,
		"/api/snapshot",
		"#/components/schemas/SnapshotIdleEvent",
		"#/components/schemas/SnapshotProcessingEvent",
		"#/components/schemas/SnapshotDoneEvent",
	); err != nil {
		log.Fatalf("flatten sse schema: %v", err)
	}

	spec, err := api.OpenAPI().MarshalJSON()
	if err != nil {
		log.Fatalf("build openapi spec: %v", err)
	}

	out, err := outputPath()
	if err != nil {
		log.Fatalf("resolve output path: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	if err := os.WriteFile(out, spec, 0o644); err != nil {
		log.Fatalf("write %s: %v", out, err)
	}

	log.Printf("done: %s", out)
}

// flattenSSESchema replaces the SSE response schema of the method operation on
// path with a oneOf of refs.
//
// huma documents an SSE response as an array of {data, event, id, retry} envelopes, but the
// generated TypeScript client applies the response schema to the `data` of a single event.
// Reducing the schema to a union of event payloads is what makes that client validate and type
// events correctly. The trade-off is that the spec drops the event names, so the payloads must
// stay distinguishable by shape alone.
func flattenSSESchema(spec *huma.OpenAPI, method, path string, refs ...string) error {
	var operation *huma.Operation
	if item := spec.Paths[path]; item != nil {
		switch method {
		case http.MethodGet:
			operation = item.Get
		case http.MethodPost:
			operation = item.Post
		}
	}

	if operation == nil {
		return fmt.Errorf("no %s operation on %s", method, path)
	}

	response := operation.Responses["200"]
	if response == nil {
		return fmt.Errorf("no 200 response on %s %s", method, path)
	}

	media := response.Content[sseMediaType]
	if media == nil {
		return fmt.Errorf("%s %s has no %s response content", method, path, sseMediaType)
	}

	oneOf := make([]*huma.Schema, 0, len(refs))
	for _, ref := range refs {
		oneOf = append(oneOf, &huma.Schema{Ref: ref})
	}

	media.Schema = &huma.Schema{OneOf: oneOf}

	return nil
}

// outputPath returns the file to write, taken from OPENAPI_OUT. It is required.
func outputPath() (string, error) {
	p := os.Getenv(outEnv)
	if p == "" {
		return "", errors.New(outEnv + " is not set")
	}

	return p, nil
}

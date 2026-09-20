package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"hump/config"
	getwebuiconfig "hump/internal/application/webui/get_webui_config"
	getwebuimetrics "hump/internal/application/webui/get_webui_metrics"
	getwebuioperations "hump/internal/application/webui/get_webui_operations"
	streamwebuiupdates "hump/internal/application/webui/stream_webui_updates"
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
		getwebuiconfig.NewController(options.Options{}, config.Configuration{}, nil),
		getwebuimetrics.NewController(nil),
		getwebuioperations.NewController(nil),
		streamwebuiupdates.NewController(nil, nil, nil),
	).Api(chi.NewMux())

	if err := flattenSSESchema(api.OpenAPI(), "/api/live", "#/components/schemas/OperationsEvent"); err != nil {
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

// flattenSSESchema replaces the SSE response schema of path with a reference to ref.
//
// huma documents an SSE response as an array of {data, event, id, retry} envelopes, but the
// generated TypeScript client applies the response schema to the `data` of a single event.
// Reducing the schema to the event payload is what makes that client validate and type events
// correctly. The trade-off is that the spec no longer describes the envelope, which only works
// while the stream carries one event name.
func flattenSSESchema(spec *huma.OpenAPI, path string, ref string) error {
	item := spec.Paths[path]
	if item == nil || item.Get == nil {
		return fmt.Errorf("no GET operation on %s", path)
	}

	response := item.Get.Responses["200"]
	if response == nil {
		return fmt.Errorf("no 200 response on GET %s", path)
	}

	media := response.Content[sseMediaType]
	if media == nil {
		return fmt.Errorf("GET %s has no %s response content", path, sseMediaType)
	}

	media.Schema = &huma.Schema{Ref: ref}

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

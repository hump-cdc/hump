package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"hump/config"
	getconfig "hump/internal/application/get_config"
	"hump/internal/controller"
	"hump/pkg/app/options"

	"github.com/go-chi/chi/v5"
)

const outEnv = "OPENAPI_OUT"

func main() {
	// Handlers are only registered, never invoked, so their dependencies can stay empty.
	api := controller.NewWebUIController(
		getconfig.NewController(options.Options{}, config.Configuration{}, nil),
	).Api(chi.NewMux())

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

// outputPath returns the file to write, taken from OPENAPI_OUT. It is required.
func outputPath() (string, error) {
	p := os.Getenv(outEnv)
	if p == "" {
		return "", errors.New(outEnv + " is not set")
	}

	return p, nil
}

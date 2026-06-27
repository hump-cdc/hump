package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"hump/internal/app/options"
	"hump/internal/types/config"
	"hump/internal/web"

	"github.com/go-chi/chi/v5"
)

const outEnv = "OPENAPI_OUT"

func main() {
	api, err := web.New(options.Options{}, config.Configuration{}, nil, nil).Api(chi.NewMux())
	if err != nil {
		log.Fatalf("setup api: %v", err)
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

// outputPath returns the file to write, taken from OPENAPI_OUT. It is required.
func outputPath() (string, error) {
	p := os.Getenv(outEnv)
	if p == "" {
		return "", errors.New(outEnv + " is not set")
	}

	return p, nil
}

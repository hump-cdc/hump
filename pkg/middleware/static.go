package middleware

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"hump/pkg/observability/log"
)

var logger = log.Scope("web")

// Static serves the embedded single-page application. Asset requests are served
// directly; unknown paths fall back to the SPA shell (index.html) so the client
// router can render them.
func Static(assetsFS embed.FS, path string) http.HandlerFunc {
	s, err := newStatic(assetsFS, path)
	if err != nil {
		logger.Error("failed to initialize static handler: %v", "err", err)
		return http.NotFoundHandler().ServeHTTP
	}

	return s.handler()
}

type static struct {
	assets fs.FS
}

type routes = map[string]struct{}

func newStatic(assetsFS embed.FS, path string) (*static, error) {
	assets, err := fs.Sub(assetsFS, path)
	if err != nil {
		return nil, err
	}

	return &static{assets: assets}, nil
}

func (s *static) handler() http.HandlerFunc {
	server := http.FileServerFS(s.assets)
	routes := s.routes()

	return func(writer http.ResponseWriter, request *http.Request) {
		if s.file(writer, request, server) {
			return
		}

		if s.index(writer, request, routes) {
			return
		}

		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

func (s *static) routes() routes {
	routes := make(routes)

	metadata, err := fs.ReadFile(s.assets, "routes.json")
	if err != nil {
		logger.Warn("web: routes.json not found")
		return routes
	}

	var paths []string
	if err := json.Unmarshal(metadata, &paths); err != nil {
		logger.Warn("web: routes.json is malformed")
		return routes
	}

	for _, p := range paths {
		routes[s.normalize(p)] = struct{}{}
	}

	return routes
}

func (s *static) index(writer http.ResponseWriter, request *http.Request, routes routes) bool {
	status := http.StatusNotFound

	route := s.normalize(request.URL.Path)
	if _, ok := routes[route]; ok || route == "/" {
		status = http.StatusOK
	}

	index, err := fs.ReadFile(s.assets, "index.html")
	if err != nil {
		return false
	}

	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)

	writer.Write(index)

	return true
}

func (s *static) file(writer http.ResponseWriter, request *http.Request, server http.Handler) bool {
	fileName := strings.TrimPrefix(request.URL.Path, "/")
	if fileName == "" || fileName == "routes.json" {
		return false
	}

	info, err := fs.Stat(s.assets, fileName)
	if err != nil || info.IsDir() {
		return false
	}

	if strings.HasPrefix(fileName, "static/") {
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	server.ServeHTTP(writer, request)

	return true
}

func (s *static) normalize(path string) string {
	if path == "/" {
		return path
	}
	return strings.TrimSuffix(path, "/")
}

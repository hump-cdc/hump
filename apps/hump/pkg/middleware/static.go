package middleware

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/http"
	"regexp"
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
	policy := s.policy()

	return func(writer http.ResponseWriter, request *http.Request) {
		if s.file(writer, request, server) {
			return
		}

		if s.index(writer, request, routes, policy) {
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

// policy builds the Content-Security-Policy for the shell, allowing each inline
// script of index.html by its hash.
func (s *static) policy() string {
	index, _ := fs.ReadFile(s.assets, "index.html") // missing: index() serves no shell anyway

	scripts := strings.Join(append([]string{"'self'"}, scriptHashes(index)...), " ")

	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + scripts,
		// UI libraries inject <style> elements and style attributes at runtime
		// without a nonce, so styles cannot be pinned by hash.
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self'",
		"font-src 'self'",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

func (s *static) index(writer http.ResponseWriter, request *http.Request, routes routes, policy string) bool {
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
	writer.Header().Set("Content-Security-Policy", policy)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)

	_, _ = writer.Write(index) // client gone; nothing to recover

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

// scriptPattern matches a <script> element, capturing its attributes and its
// content exactly as written between the tags.
var scriptPattern = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script\s*>`)

// srcPattern matches a src attribute among a <script> element's attributes.
var srcPattern = regexp.MustCompile(`(?i)(?:^|\s)src(?:\s*=|\s|$)`)

// scriptHashes returns a 'sha256-…' CSP source for every inline <script> in html,
// in document order.
func scriptHashes(html []byte) []string {
	var hashes []string

	for _, match := range scriptPattern.FindAllSubmatch(html, -1) {
		if srcPattern.Match(match[1]) {
			continue
		}

		sum := sha256.Sum256(match[2])
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}

	return hashes
}

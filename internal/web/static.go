package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// staticHandler serves files from the embedded SPA asset tree. Requests whose
// path does not match an existing file fall back to index.html so that
// client-side routes resolve, while requests for hashed static bundles get a
// long-lived immutable cache header. Unknown paths that are not declared
// client-side routes are served with HTTP 404 (still rendering the SPA shell so
// the in-app NotFound page is shown).
func staticHandler(assets fs.FS) http.HandlerFunc {
	fileServer := http.FileServerFS(assets)
	routes := loadRoutes(assets)

	return func(rw http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")

		// Serve real files (and directories) directly. fs.Stat fails for
		// missing files and for client-side route paths (which map to no file),
		// both of which fall through to the SPA shell below. routes.json is a
		// server-internal manifest and must never be exposed.
		if name != "" && name != "routes.json" {
			if info, err := fs.Stat(assets, name); err == nil && !info.IsDir() {
				// Content-addressed bundles never change for a given URL.
				if strings.HasPrefix(name, "static/") {
					rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(rw, r)
				return
			}
		}

		// Root and client-side routes render the SPA entry point with HTTP 200;
		// anything else is a genuine miss and gets HTTP 404 while still serving
		// the shell so the in-app NotFound page renders. Serving index.html
		// no-cache (rather than via the file server) ensures clients always
		// fetch HTML that references the current hashed bundles.
		path := normalizeRoute(r.URL.Path)
		status := http.StatusNotFound
		if _, ok := routes[path]; ok || path == "/" {
			status = http.StatusOK
		}
		serveIndex(rw, r, assets, status)
	}
}

// loadRoutes reads the build-generated routes.json manifest into a set of valid
// client-side route paths. A missing or malformed manifest yields an empty set,
// so the handler degrades to serving index.html for every fallback (the prior
// behaviour) rather than failing.
func loadRoutes(assets fs.FS) map[string]struct{} {
	routes := make(map[string]struct{})

	data, err := fs.ReadFile(assets, "routes.json")
	if err != nil {
		logger.Warn("web: routes.json not found; not-found pages will return HTTP 200")
		return routes
	}

	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		logger.Warn("web: routes.json is malformed; not-found pages will return HTTP 200")
		return routes
	}

	for _, p := range paths {
		routes[normalizeRoute(p)] = struct{}{}
	}

	return routes
}

// normalizeRoute canonicalizes a URL path for comparison against the manifest,
// trimming a trailing slash so that "/stats/" matches "/stats" while "/" is
// preserved.
func normalizeRoute(path string) string {
	if path == "/" {
		return path
	}
	return strings.TrimSuffix(path, "/")
}

// serveIndex writes the SPA entry point for client-side routes with the given
// status code. A 200 response goes through http.ServeFileFS to preserve
// Last-Modified and range handling; other statuses (e.g. 404) require writing
// the body directly, since ServeFileFS always responds with 200.
func serveIndex(rw http.ResponseWriter, r *http.Request, assets fs.FS, status int) {
	rw.Header().Set("Cache-Control", "no-cache")

	if status == http.StatusOK {
		http.ServeFileFS(rw, r, assets, "index.html")
		return
	}

	data, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(status)
	_, _ = rw.Write(data)
}

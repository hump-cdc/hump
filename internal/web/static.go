package web

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

// staticHandler serves files from the embedded SPA asset tree. Requests whose
// path does not match an existing file fall back to index.html so that
// client-side routes resolve, while requests for hashed static bundles get a
// long-lived immutable cache header.
func staticHandler(assets fs.FS) http.HandlerFunc {
	fileServer := http.FileServerFS(assets)

	return func(rw http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")

		// Root and unknown paths render the SPA entry point. Serving it
		// no-cache (rather than via the file server) ensures clients always
		// fetch HTML that references the current hashed bundles.
		if name == "" {
			serveIndex(rw, r, assets)
			return
		}
		if _, err := fs.Stat(assets, name); errors.Is(err, fs.ErrNotExist) {
			serveIndex(rw, r, assets)
			return
		}

		// Content-addressed bundles never change for a given URL.
		if strings.HasPrefix(name, "static/") {
			rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		fileServer.ServeHTTP(rw, r)
	}
}

// serveIndex writes the SPA entry point for client-side routes.
func serveIndex(rw http.ResponseWriter, r *http.Request, assets fs.FS) {
	rw.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(rw, r, assets, "index.html")
}

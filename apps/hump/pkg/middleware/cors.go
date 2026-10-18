// Package middleware provides HTTP middleware and handlers for the web server:
// CORS handling and the embedded single-page application static handler.
package middleware

import (
	"net/http"
	"os"
	"strings"

	chicors "github.com/go-chi/cors"
)

// Cors builds a CORS middleware from the comma-separated CORS environment
// variable. Without configured origins it returns a pass-through.
func Cors() func(http.Handler) http.Handler {
	origins := make([]string, 0)
	for origin := range strings.SplitSeq(strings.TrimSpace(os.Getenv("CORS")), ",") {
		if o := strings.TrimSpace(origin); o != "" {
			origins = append(origins, o)
		}
	}

	if len(origins) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return chicors.Handler(chicors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

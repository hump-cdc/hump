package middleware

import (
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// SnapshotMutations requires JSON and a trusted browser origin for snapshot start and stop.
// Requests without Origin remain available to non-browser clients.
func SnapshotMutations() func(http.Handler) http.Handler {
	origins := make(map[string]bool)
	for entry := range strings.SplitSeq(os.Getenv("CORS"), ",") {
		if origin := strings.TrimSpace(entry); validOrigin(origin) {
			origins[origin] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || (r.URL.Path != "/api/snapshot/start" && r.URL.Path != "/api/snapshot/stop") {
				next.ServeHTTP(w, r)
				return
			}

			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
				http.Error(w, "application/json required", http.StatusUnsupportedMediaType)
				return
			}

			if values, present := r.Header["Origin"]; present {
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				if len(values) != 1 || !validOrigin(values[0]) || (values[0] != scheme+"://"+r.Host && !origins[values[0]]) {
					http.Error(w, "origin forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validOrigin accepts only explicit HTTP(S) origins without URL suffixes or wildcards.
func validOrigin(origin string) bool {
	if strings.ContainsAny(origin, "* ,\t\r\n") {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
		u.Hostname() == "" || u.User != nil || origin != u.Scheme+"://"+u.Host {
		return false
	}
	if strings.ContainsAny(u.Hostname(), ":[]") || strings.HasPrefix(u.Host, "[") {
		ip, err := netip.ParseAddr(u.Hostname())
		if err != nil || !ip.Is6() || ip.Zone() != "" || !strings.HasPrefix(u.Host, "[") {
			return false
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return false
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return false
		}
	}
	return true
}

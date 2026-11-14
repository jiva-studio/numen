// Package access decides which HTTP requests may reach the MCP server: a request needs the bearer
// token, and a request that names an origin needs a loopback one.
package access

import (
	"net"
	"net/http"
	"net/url"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// IsLoopbackOrigin reports whether the origin is an http or https URL whose host is localhost or a
// loopback address.
func IsLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// RequireToken answers 403 to a request from an origin outside the machine and 401 to a request
// without the token, and passes every other request to next.
func RequireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origins, hasOrigin := r.Header["Origin"]; hasOrigin && (len(origins) != 1 || !IsLoopbackOrigin(origins[0])) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !server.HasBearerToken(r.Header, token) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

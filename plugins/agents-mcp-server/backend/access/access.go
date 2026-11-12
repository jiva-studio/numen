// Package access decides which HTTP requests may reach the MCP server: a request needs the bearer
// token, and a request that names an origin needs a loopback one.
package access

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// HasValidToken reports whether the request carries the token as a bearer credential. An empty
// token matches no request.
func HasValidToken(token string, r *http.Request) bool {
	if token == "" {
		return false
	}
	bearer, isBearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return isBearer && subtle.ConstantTimeCompare([]byte(bearer), []byte(token)) == 1
}

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
		if !HasValidToken(token, r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

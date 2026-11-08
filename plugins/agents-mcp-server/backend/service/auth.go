package service

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func hasValidToken(token string, r *http.Request) bool {
	if token == "" {
		return false
	}
	bearer, isBearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return isBearer && subtle.ConstantTimeCompare([]byte(bearer), []byte(token)) == 1
}

func isLoopbackOrigin(origin string) bool {
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

func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origins, hasOrigin := r.Header["Origin"]; hasOrigin && (len(origins) != 1 || !isLoopbackOrigin(origins[0])) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !hasValidToken(token, r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

package server

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

// ErrUnauthorized indicates missing or mismatched authentication credentials.
var ErrUnauthorized = errors.New("unauthorized")

// HasBearerToken reports whether the Authorization header carries the token as a bearer credential,
// compared in constant time. An empty token matches no header.
func HasBearerToken(header http.Header, token string) bool {
	if token == "" {
		return false
	}
	bearer, isBearer := strings.CutPrefix(header.Get("Authorization"), "Bearer ")
	return isBearer && subtle.ConstantTimeCompare([]byte(bearer), []byte(token)) == 1
}

// RequireToken wraps next so that a request without the bearer token is answered 401.
func RequireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !HasBearerToken(r.Header, token) {
			http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

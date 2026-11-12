// Package localendpoint holds what every HTTP endpoint on the loopback interface shares: a random
// token made for one start of the process, and a check that a request presents it. Any process on
// the machine can connect to a loopback port, so a port that serves the vault answers only a caller
// that was given the token.
package localendpoint

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

const (
	tokenBytes   = 32
	bearerPrefix = "Bearer "
)

// NewToken returns a random token of 256 bits, hex encoded.
func NewToken() (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("create a token for a local endpoint: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// IsTokenEqual reports whether the presented token is the expected one, in time that does not
// depend on where they differ. An empty expected token matches nothing.
func IsTokenEqual(expected, presented string) bool {
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) == 1
}

// HasBearerToken reports whether the request carries the token as a bearer credential.
func HasBearerToken(token string, r *http.Request) bool {
	presented, isBearer := strings.CutPrefix(strings.TrimSpace(r.Header.Get("Authorization")), bearerPrefix)
	return isBearer && IsTokenEqual(token, presented)
}

// RequireBearerToken answers 401 to every request that does not carry the token as a bearer
// credential and passes the others to next.
func RequireBearerToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !HasBearerToken(token, r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

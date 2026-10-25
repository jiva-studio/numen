package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

// createRandomToken generates a 32-byte hex-encoded random authentication token.
func createRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hasValidToken reports whether the request presents the expected authentication token.
func hasValidToken(token string, r *http.Request) bool {
	if token == "" {
		return true
	}
	expected := []byte(token)

	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		given := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(given), expected) == 1 {
			return true
		}
	}

	if queryToken := r.URL.Query().Get("token"); queryToken != "" {
		if subtle.ConstantTimeCompare([]byte(queryToken), expected) == 1 {
			return true
		}
	}

	return false
}

// requireToken validates authentication on incoming requests.
func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		if !hasValidToken(token, r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="numen"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

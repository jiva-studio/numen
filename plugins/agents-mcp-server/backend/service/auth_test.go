package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRandomToken(t *testing.T) {
	t.Parallel()

	token1, err := createRandomToken()
	if err != nil {
		t.Fatalf("unexpected error creating token: %v", err)
	}
	if len(token1) != 64 {
		t.Fatalf("expected 64 character hex token, got %d", len(token1))
	}

	token2, err := createRandomToken()
	if err != nil {
		t.Fatalf("unexpected error creating second token: %v", err)
	}
	if token1 == token2 {
		t.Fatal("expected distinct random tokens")
	}
}

func TestHasValidToken(t *testing.T) {
	t.Parallel()

	token := "test-secret-token"

	tests := []struct {
		name       string
		authHeader string
		queryToken string
		expected   bool
	}{
		{
			name:       "valid bearer header",
			authHeader: "Bearer test-secret-token",
			expected:   true,
		},
		{
			name:       "invalid bearer header",
			authHeader: "Bearer wrong-token",
			expected:   false,
		},
		{
			name:       "valid query token",
			queryToken: "test-secret-token",
			expected:   true,
		},
		{
			name:       "invalid query token",
			queryToken: "wrong-token",
			expected:   false,
		},
		{
			name:     "no token provided",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			url := "http://localhost/mcp"
			if tc.queryToken != "" {
				url += "?token=" + tc.queryToken
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			if got := hasValidToken(token, req); got != tc.expected {
				t.Fatalf("hasValidToken() = %v, want %v", got, tc.expected)
			}
		})
	}

	// Empty configured token allows everything.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/mcp", nil)
	if !hasValidToken("", req) {
		t.Fatal("expected empty token to allow access")
	}
}

func TestRequireToken(t *testing.T) {
	t.Parallel()

	token := "secret"
	innerCalled := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := requireToken(token, inner)

	// Options preflight
	reqOpts := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/mcp", nil)
	recOpts := httptest.NewRecorder()
	handler.ServeHTTP(recOpts, reqOpts)
	if recOpts.Code != http.StatusOK {
		t.Fatalf("OPTIONS returned status %d, want 200", recOpts.Code)
	}

	// Unauthorized request
	reqUnauth := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/mcp", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized returned %d, want 401", recUnauth.Code)
	}
	if innerCalled {
		t.Fatal("inner handler called when unauthorized")
	}

	// Authorized request
	reqAuth := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/mcp", nil)
	reqAuth.Header.Set("Authorization", "Bearer secret")
	recAuth := httptest.NewRecorder()
	handler.ServeHTTP(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Fatalf("authorized returned %d, want 200", recAuth.Code)
	}
	if !innerCalled {
		t.Fatal("inner handler not called when authorized")
	}
}

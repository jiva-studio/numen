package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLegacySSERoutesAreUnknown(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{Token: "route-token"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/sse", "/sse/", "/sse/sub", "/messages", "/messages/", "/messages/sub"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
		req.Header.Set("Authorization", "Bearer route-token")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("POST %s returned %d, want 404", path, rec.Code)
		}
	}
}

func TestStreamableRouteStaysWhenTheLegacyRoutesAreUnknown(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{Token: "route-token"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/mcp", "/mcp/"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, path, nil)
		req.Header.Set("Authorization", "Bearer route-token")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("PUT %s returned %d, want 405", path, rec.Code)
		}
	}
}

func TestReadmeNamesNoLegacyRoute(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, removed := range []string{"/sse", "/messages", "SSE"} {
		if strings.Contains(string(readme), removed) {
			t.Errorf("README names %q", removed)
		}
	}
}

package agenttoolsbridge

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
	`{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"orchard","version":"1"}}}`

func serveOriginEndpoint(t *testing.T) *endpoint {
	t.Helper()
	served, err := serveEndpoint(t.Context(), nil, callScope{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := served.close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	return served
}

func postInitialize(t *testing.T, served *endpoint, token string, origins ...string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, served.url, strings.NewReader(initializeBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for _, origin := range origins {
		req.Header.Add("Origin", origin)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		t.Error(err)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}
	return res.StatusCode
}

func TestARequestFromAnOriginOutsideTheMachineGets403FromTheRunEndpoint(t *testing.T) {
	served := serveOriginEndpoint(t)

	for _, origin := range []string{
		"null",
		"http://example.com",
		"https://example.com:3333",
		"http://127.0.0.1.example.com",
		"http://localhost.example.com",
		"http://192.168.1.20:3333",
		"ftp://localhost",
		"localhost",
	} {
		if status := postInitialize(t, served, served.token, origin); status != http.StatusForbidden {
			t.Errorf("an initialize with the token from the origin %s returned %d, want 403", origin, status)
		}
		if status := postInitialize(t, served, "", origin); status != http.StatusForbidden {
			t.Errorf("an initialize without the token from the origin %s returned %d, want 403", origin, status)
		}
	}
}

func TestARequestWithTwoOriginsGets403FromTheRunEndpoint(t *testing.T) {
	served := serveOriginEndpoint(t)

	if status := postInitialize(t, served, served.token, "http://localhost", "http://127.0.0.1"); status != http.StatusForbidden {
		t.Errorf("an initialize with the token and two origins returned %d, want 403", status)
	}
}

func TestARequestFromALoopbackOriginOrNoOriginIsCheckedForTheTokenOnly(t *testing.T) {
	served := serveOriginEndpoint(t)

	for _, origins := range [][]string{
		nil,
		{"http://localhost"},
		{"http://localhost:5173"},
		{"https://localhost:8443"},
		{"http://127.0.0.1:5173"},
		{"http://127.8.9.10"},
		{"http://[::1]:3333"},
	} {
		if status := postInitialize(t, served, served.token, origins...); status != http.StatusOK {
			t.Errorf("an initialize with the token from the origins %v returned %d, want 200", origins, status)
		}
		if status := postInitialize(t, served, "", origins...); status != http.StatusUnauthorized {
			t.Errorf("an initialize without the token from the origins %v returned %d, want 401", origins, status)
		}
		if status := postInitialize(t, served, "another", origins...); status != http.StatusUnauthorized {
			t.Errorf("an initialize with another token from the origins %v returned %d, want 401", origins, status)
		}
	}
}

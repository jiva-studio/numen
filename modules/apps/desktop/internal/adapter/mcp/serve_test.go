package mcp_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/mcp"
)

// The port is open on this machine, so what stands in front of it is the whole
// of who may reach somebody's notes.
func TestNothingReachesTheVaultWithoutTheToken(t *testing.T) {
	_, core := served(t)
	endpoint, err := mcp.ServeHTTP(t.Context(), "127.0.0.1:0", "the-token", core, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { endpoint.Close(context.Background()) })

	for name, header := range map[string]string{
		"nothing":       "",
		"another":       "Bearer someone-elses",
		"a prefix":      "Bearer the-toke",
		"the wrong way": "the-token",
	} {
		t.Run(name, func(t *testing.T) {
			res := ask(t, endpoint.URL, header, "")
			if res.StatusCode != http.StatusUnauthorized {
				t.Errorf("want 401, got %s", res.Status)
			}
		})
	}
}

// A page open in a browser can reach a port on this machine, and is the one
// caller that arrives without being invited.
func TestAPageInABrowserIsTurnedAway(t *testing.T) {
	_, core := served(t)
	endpoint, err := mcp.ServeHTTP(t.Context(), "127.0.0.1:0", "the-token", core, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { endpoint.Close(context.Background()) })

	res := ask(t, endpoint.URL, "Bearer the-token", "https://example.com")
	if res.StatusCode == http.StatusOK {
		t.Errorf("a cross-origin request was answered: %s", res.Status)
	}
}

func TestLocalKnowsWhichAddressesLeaveTheMachine(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:7717": true,
		"localhost:7717": true,
		"[::1]:7717":     true,
		"0.0.0.0:7717":   false,
		"192.168.1.4:80": false,
		"nonsense":       false,
	} {
		if got := mcp.Local(addr); got != want {
			t.Errorf("%s: want %v, got %v", addr, want, got)
		}
	}
}

func TestAServerNeedsSomethingToAskFor(t *testing.T) {
	_, core := served(t)
	if _, err := mcp.ServeHTTP(t.Context(), "127.0.0.1:0", "", core, nil); err == nil {
		t.Fatal("a server with no token is open to everything on the machine")
	}
}

// ask sends the smallest thing the endpoint will look at.
func ask(t *testing.T, url, authorization, origin string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url,
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

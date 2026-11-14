package mcpendpoint

import (
	"crypto/rand"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func dialEndpoint(t *testing.T, served *Endpoint) net.Conn {
	t.Helper()
	address, err := url.Parse(served.url)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", address.Host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Log(err)
		}
	})
	return conn
}

func waitFor(t *testing.T, isReady func() bool) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for !isReady() {
		select {
		case <-deadline:
			t.Fatal("the endpoint did not reach the expected state")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestCloseDoesNotWaitForAConnectionThatSentNoRequest(t *testing.T) {
	served, err := serve(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	dialEndpoint(t, served)
	waitFor(t, func() bool {
		served.statesMu.Lock()
		defer served.statesMu.Unlock()
		return len(served.states) > 0
	})

	closed := make(chan error, 1)
	go func() { closed <- served.Close(t.Context()) }()
	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("close reported %v for a connection with no request", err)
		}
	case <-time.After(shutdownLimit / 2):
		t.Errorf("close took %v or longer", shutdownLimit/2)
	}
}

func TestCloseReportsAnInterruptedRequest(t *testing.T) {
	served, err := serve(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	served.limit = 100 * time.Millisecond
	conn := dialEndpoint(t, served)
	if _, err := conn.Write([]byte("POST /mcp HTTP/1.1\r\nHost: stand-in\r\nContent-Length: 100\r\n\r\n{")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return served.getActiveCount() > 0 })

	if err := served.Close(t.Context()); err == nil {
		t.Error("close reported nothing for a request it cut")
	}
}

func TestEndpointConfigurationConnectionTrackingAndTools(t *testing.T) {
	if shutdownLimit != 2*time.Second || shutdownPoll != 10*time.Millisecond || tokenBytes != 32 {
		t.Fatalf("constants = (%v, %v, %d)", shutdownLimit, shutdownPoll, tokenBytes)
	}
	if _, err := serveOn(t.Context(), iotest.ErrReader(errors.New("no entropy")), "127.0.0.1:0", nil, nil); err == nil {
		t.Error("expected error when token reader fails")
	}
	if _, err := serveOn(t.Context(), rand.Reader, "127.0.0.1:invalid", nil, nil); err == nil {
		t.Error("expected error when listen address is invalid")
	}

	listed := []*tools.ToolInfo{
		{Name: "note_read", Effect: tools.ReadOnly},
		{Name: "note_edit", Effect: tools.ReadWrite},
	}
	servedTools, err := (Server{}).Serve(t.Context(), listed, nil)
	if err != nil {
		t.Fatal(err)
	}
	served, ok := servedTools.(*Endpoint)
	if !ok {
		t.Fatalf("expected *Endpoint, got %T", servedTools)
	}
	if len(served.GetToken()) != 64 || !strings.HasSuffix(served.GetURL(), "/mcp") {
		t.Errorf("token len = %d, url = %q", len(served.GetToken()), served.GetURL())
	}
	if served.server.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 10s", served.server.ReadHeaderTimeout)
	}

	// Verify both tools in listed are registered on the MCP server.
	sessionID := initializeSession(t, served)
	toolsBody := postMCP(t, served, sessionID, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if !strings.Contains(toolsBody, `"note_read"`) || !strings.Contains(toolsBody, `"note_edit"`) {
		t.Errorf("tools/list body = %s, want both note_read and note_edit", toolsBody)
	}

	// Verify trackConnection removes both Closed and Hijacked connections.
	clientConn, serverConn := net.Pipe()
	defer func() {
		if err := errors.Join(clientConn.Close(), serverConn.Close()); err != nil {
			t.Log(err)
		}
	}()
	// The requests above leave their own kept-alive connection tracked, so the counts are relative to it.
	before := getTrackedCount(served)
	served.trackConnection(clientConn, http.StateNew)
	if got := getTrackedCount(served); got != before+1 {
		t.Errorf("tracked connections = %d, want %d", got, before+1)
	}
	served.trackConnection(clientConn, http.StateClosed)
	if got := getTrackedCount(served); got != before {
		t.Errorf("tracked connections after StateClosed = %d, want %d", got, before)
	}
	served.trackConnection(clientConn, http.StateNew)
	served.trackConnection(clientConn, http.StateHijacked)
	if got := getTrackedCount(served); got != before {
		t.Errorf("tracked connections after StateHijacked = %d, want %d", got, before)
	}

	// Verify Close waits for an active request that finishes within shutdownLimit.
	conn := dialEndpoint(t, served)
	if _, err := conn.Write([]byte("POST /mcp HTTP/1.1\r\nHost: stand-in\r\nContent-Length: 2\r\n\r\n{")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return served.getActiveCount() > 0 })
	var isFinished atomic.Bool
	go func() {
		time.Sleep(25 * time.Millisecond)
		isFinished.Store(true)
		if _, err := conn.Write([]byte("}")); err != nil {
			t.Log(err)
		}
	}()
	if err := served.Close(t.Context()); err != nil {
		t.Errorf("Close with finishing request = %v, want nil", err)
	}
	if !isFinished.Load() {
		t.Error("Close returned before the active request finished")
	}
}

func initializeSession(t *testing.T, served *Endpoint) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, served.GetURL(), strings.NewReader(initializeBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+served.GetToken())
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
	return res.Header.Get("Mcp-Session-Id")
}

func postMCP(t *testing.T, served *Endpoint, sessionID, body string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, served.GetURL(), strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+served.GetToken())
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}
	return string(raw)
}

func getTrackedCount(served *Endpoint) int {
	served.statesMu.Lock()
	defer served.statesMu.Unlock()
	return len(served.states)
}

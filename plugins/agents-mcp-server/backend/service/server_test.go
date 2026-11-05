package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServerLifecycleAndToolCall(t *testing.T) {
	t.Parallel()

	hostClient := &fakeToolClient{
		tools: []*tools.ToolInfo{
			{
				Name:        "note_get",
				Title:       "Get Note",
				Description: "Gets a note by path",
				InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
				Effect:      tools.ReadOnly,
			},
		},
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text: "note text for test",
			},
		},
	}

	cfg := Config{
		Addr:  "127.0.0.1:0",
		Token: "test-token-123",
	}

	srv, err := NewServer(cfg, hostClient)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if got := srv.GetAggregator().GetRegisteredTools(); len(got) != 1 || got[0] != "note_get" {
		t.Fatalf("expected initial SyncTools in NewServer, got %v", got)
	}
	if srv.GetURL() != "" || srv.GetHTTPURL() != "" {
		t.Fatal("expected empty URLs before Start")
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start failed: %v", err)
	}
	defer func() {
		if err := srv.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	mcpURL := srv.GetHTTPURL()
	if mcpURL == "" {
		t.Fatal("expected non-empty streamable HTTP URL")
	}

	assertUnauthorized(t, mcpURL)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	session := connectStreamableClient(ctx, t, mcpURL, "test-token-123")
	callNoteGetTool(ctx, t, session)
	assertDynamicDiscovery(ctx, t, session, hostClient)

	cleanup.Close(t, session)

	if err := srv.Close(t.Context()); err != nil {
		t.Fatalf("srv.Close failed: %v", err)
	}
}

func assertUnauthorized(t *testing.T, url string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func callNoteGetTool(ctx context.Context, t *testing.T, session *sdk.ClientSession) {
	t.Helper()

	listResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("session.ListTools failed: %v", err)
	}
	if len(listResult.Tools) != 1 || listResult.Tools[0].Name != "note_get" {
		t.Fatalf("unexpected tools returned: %+v", listResult.Tools)
	}

	callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "note_get",
		Arguments: map[string]any{
			"path": "hello.md",
		},
	})
	if err != nil {
		t.Fatalf("session.CallTool failed: %v", err)
	}
	if callResult.IsError || len(callResult.Content) == 0 {
		t.Fatal("expected successful tool call with content")
	}
	textContent, ok := callResult.Content[0].(*sdk.TextContent)
	if !ok || textContent.Text != "note text for test" {
		t.Fatalf("unexpected call result content: %v", callResult.Content[0])
	}
}

func assertDynamicDiscovery(ctx context.Context, t *testing.T, session *sdk.ClientSession, hostClient *fakeToolClient) {
	t.Helper()

	hostClient.mu.Lock()
	hostClient.tools = append(hostClient.tools, &tools.ToolInfo{
		Name:        "new_vault_tool",
		Title:       "New Tool",
		Description: "Newly registered tool",
		InputSchema: []byte(`{"type":"object"}`),
	})
	hostClient.mu.Unlock()

	listResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("second ListTools failed: %v", err)
	}
	if len(listResult.Tools) != 2 {
		t.Fatalf("expected 2 tools after dynamic addition, got %d", len(listResult.Tools))
	}
}

func connectStreamableClient(ctx context.Context, t *testing.T, mcpURL, token string) *sdk.ClientSession {
	t.Helper()
	authClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", "Bearer "+token)
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
	clientTransport := &sdk.StreamableClientTransport{
		Endpoint:   mcpURL,
		HTTPClient: authClient,
	}
	mcpClient := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("mcpClient.Connect failed: %v", err)
	}
	return session
}

type failingListener struct {
	err      error
	closeErr error
	onClose  func()
}

func (l *failingListener) Accept() (net.Conn, error) {
	return nil, l.err
}

func (l *failingListener) Close() error {
	if l.onClose != nil {
		l.onClose()
	}
	return l.closeErr
}

func (l *failingListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 54321}
}

func TestServerOptionsAndRoutes(t *testing.T) {
	t.Parallel()

	srvOpts := newServerOptions()
	if srvOpts == nil || srvOpts.Instructions != "Numén notes and plugins tools server." {
		t.Fatalf("unexpected server options: %+v", srvOpts)
	}

	streamOpts := newStreamableOptions()
	if streamOpts == nil || streamOpts.SessionTimeout != 30*time.Minute {
		t.Fatalf("unexpected streamable options: %+v", streamOpts)
	}

	srv, err := NewServer(Config{Token: "route-token"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if srv.httpServer.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 10s", srv.httpServer.ReadHeaderTimeout)
	}

	paths := []string{"/mcp", "/mcp/sub"}
	for _, path := range paths {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, path, nil)
		req.Header.Set("Authorization", "Bearer route-token")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("PUT %s returned %d, want 405 Method Not Allowed", path, rec.Code)
		}
	}
}

func TestServerStartRebindAndClose(t *testing.T) {
	t.Parallel()

	// Start with invalid address fails.
	badSrv, err := NewServer(Config{Addr: "127.0.0.1:invalid", Token: "t"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := badSrv.Start(); err == nil {
		t.Fatal("expected Start with invalid address to fail")
	}

	// Start with empty address defaults to 127.0.0.1:0.
	srv, err := NewServer(Config{Addr: "", Token: "tok-1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := srv.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	url1 := srv.GetHTTPURL()
	if !strings.HasPrefix(url1, "http://127.0.0.1:") {
		t.Fatalf("expected loopback streamable HTTP URL, got %s", url1)
	}
	addr1 := srv.actualAddr

	// Rebind with empty Addr only updates Token without restarting listener.
	if err := srv.Rebind(t.Context(), Config{Addr: "", Token: "tok-2"}); err != nil {
		t.Fatalf("Rebind empty addr failed: %v", err)
	}
	if srv.GetToken() != "tok-2" || srv.GetHTTPURL() != url1 {
		t.Fatalf("expected token tok-2 and unchanged URL %s, got token=%s url=%s", url1, srv.GetToken(), srv.GetHTTPURL())
	}

	// Reserve a second port to rebind to.
	probeLn, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr2 := probeLn.Addr().String()
	if err := probeLn.Close(); err != nil {
		t.Fatal(err)
	}

	// Rebind to addr2 closes the old listener on addr1 and starts serving on addr2.
	oldHTTPServer := srv.httpServer
	if err := srv.Rebind(t.Context(), Config{Addr: addr2, Token: "tok-3"}); err != nil {
		t.Fatalf("Rebind to %s failed: %v", addr2, err)
	}
	if srv.actualAddr != addr2 || srv.GetToken() != "tok-3" {
		t.Fatalf("unexpected state after Rebind: addr=%s token=%s", srv.actualAddr, srv.GetToken())
	}
	if srv.listener == nil || srv.listener.Addr().String() != addr2 {
		t.Fatalf("expected srv.listener to be updated to %s, got %v", addr2, srv.listener)
	}
	if srv.httpServer == oldHTTPServer {
		t.Fatal("expected srv.httpServer to be replaced on Rebind")
	}

	// Verify old listener on addr1 was closed.
	recheckLn, err := new(net.ListenConfig).Listen(t.Context(), "tcp", addr1)
	if err != nil {
		t.Fatalf("expected old listener on %s to be closed: %v", addr1, err)
	}
	cleanup.Close(t, recheckLn)

	// Rebind with empty Addr when s.config.Addr is non-empty (addr2) only updates Token.
	if err := srv.Rebind(t.Context(), Config{Addr: "", Token: "tok-3b"}); err != nil {
		t.Fatalf("Rebind empty addr after addr2 failed: %v", err)
	}
	if srv.actualAddr != addr2 || srv.GetToken() != "tok-3b" {
		t.Fatalf("expected addr %s and token tok-3b, got addr=%s token=%s", addr2, srv.actualAddr, srv.GetToken())
	}

	// Rebind with the identical address addr2 keeps the existing listener.
	lnBeforeSameRebind := srv.listener
	if err := srv.Rebind(t.Context(), Config{Addr: addr2, Token: "tok-4"}); err != nil {
		t.Fatalf("Rebind same addr failed: %v", err)
	}
	if srv.listener != lnBeforeSameRebind || srv.GetToken() != "tok-4" {
		t.Fatal("expected listener to be unchanged when rebinding to same address")
	}

	// Rebind to an invalid address on an open server closes oldListener and fails on Listen.
	if err := srv.Rebind(t.Context(), Config{Addr: "127.0.0.1:invalid", Token: "tok-bad"}); err == nil {
		t.Fatal("expected Rebind to invalid address on open server to fail")
	}

	// Subsequent Rebind when oldListener is already closed (returning net.ErrClosed) succeeds.
	if err := srv.Rebind(t.Context(), Config{Addr: "127.0.0.1:0", Token: "tok-after-closed"}); err != nil {
		t.Fatalf("expected Rebind after pre-closed listener to succeed, got %v", err)
	}

	// Close after Rebind succeeds without reporting net.ErrClosed from the old listener.
	if err := srv.Close(t.Context()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Rebind on a closed server returns nil without attempting to listen.
	if err := srv.Rebind(t.Context(), Config{Addr: "127.0.0.1:invalid", Token: "tok-5"}); err != nil {
		t.Fatalf("expected Rebind on closed server to return nil, got %v", err)
	}

	// Serve on a closed server returns an error.
	if err := srv.Serve(&failingListener{err: net.ErrClosed}); err == nil {
		t.Fatal("expected Serve on closed server to fail")
	}
}

func TestServerRebindEdgeCases(t *testing.T) {
	t.Parallel()

	// Rebind returns error when closing oldListener fails with an error other than net.ErrClosed.
	srvCloseErr, err := NewServer(Config{Addr: "", Token: "t"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	closeBoom := errors.New("listener close failure")
	srvCloseErr.listener = &failingListener{err: net.ErrClosed, closeErr: closeBoom}
	if err := srvCloseErr.Rebind(t.Context(), Config{Addr: "127.0.0.1:0", Token: "t2"}); !errors.Is(err, closeBoom) {
		t.Fatalf("expected closeBoom from Rebind, got %v", err)
	}

	// Rebind closes new listener and returns if server closes concurrently during Rebind.
	srvRace, err := NewServer(Config{Addr: "", Token: "t"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	srvRace.listener = &failingListener{
		err: net.ErrClosed,
		onClose: func() {
			if err := srvRace.Close(t.Context()); err != nil {
				t.Error(err)
			}
		},
	}
	if err := srvRace.Rebind(t.Context(), Config{Addr: "127.0.0.1:0", Token: "t2"}); err != nil {
		t.Fatalf("expected nil error when closed during Rebind, got %v", err)
	}
	if srvRace.actualAddr != "" {
		t.Fatalf("expected actualAddr to remain empty when closed during Rebind, got %s", srvRace.actualAddr)
	}
}

func TestServerServeErrorAndIdempotentClose(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{Token: "tok"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	acceptErr := errors.New("accept failure")
	srv.serveHTTP(srv.httpServer, &failingListener{err: acceptErr})

	if err := srv.Close(t.Context()); !errors.Is(err, acceptErr) {
		t.Fatalf("expected Close to return %v, got %v", acceptErr, err)
	}

	// Second Close on already-closed server returns nil and unlocks mutex.
	if err := srv.Close(t.Context()); err != nil {
		t.Fatalf("expected second Close to return nil, got %v", err)
	}
	if got := srv.GetToken(); got != "tok" {
		t.Fatalf("GetToken() = %q, want tok", got)
	}
}

func TestServerCloseClosesConnectionWithoutRequest(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{Addr: "127.0.0.1:0", Token: "tok"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	addr := srv.actualAddr

	silentConn, err := new(net.Dialer).DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, silentConn)

	// The server accepts connections in order, so an answer on a later connection means it holds silentConn.
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if err := srv.Close(t.Context()); err != nil {
		t.Fatalf("Close with a connection that sent no request: %v", err)
	}
}

// isPeerClosed reports whether the end of a net.Pipe opposite to peer has been closed.
func isPeerClosed(t *testing.T, peer net.Conn) bool {
	t.Helper()
	err := peer.SetReadDeadline(time.Unix(1, 0))
	if errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	_, err = peer.Read(make([]byte, 1))
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("read from an open pipe: got %v, want a deadline error", err)
	}
	return false
}

// waitPeerClosed fails the test with msg unless the end of a net.Pipe opposite to peer is closed
// within five seconds.
func waitPeerClosed(t *testing.T, peer net.Conn, msg string) {
	t.Helper()
	err := peer.SetReadDeadline(clock.System{}.Now().Add(5 * time.Second))
	if errors.Is(err, io.ErrClosedPipe) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	_, err = peer.Read(make([]byte, 1))
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
		return
	}
	t.Fatalf("%s: read from the peer returned %v", msg, err)
}

func TestServerCloseKeepsConnectionsThatSentRequest(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{Token: "tok"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	silentConn, silentPeer := net.Pipe()
	activeConn, activePeer := net.Pipe()
	for _, c := range []net.Conn{silentConn, silentPeer, activeConn, activePeer} {
		cleanup.CloseAfter(t, c)
	}

	srv.trackConn(silentConn, http.StateNew)
	srv.trackConn(activeConn, http.StateNew)
	srv.trackConn(activeConn, http.StateActive)
	if err := srv.Close(t.Context()); err != nil {
		t.Fatal(err)
	}

	waitPeerClosed(t, silentPeer, "Close left open a connection that sent no request")
	// The connections are closed in one pass under s.mu, so the pass has ended once the lock is free.
	srv.mu.Lock()
	tracked := len(srv.newConns)
	srv.mu.Unlock()
	if tracked != 0 {
		t.Fatalf("Close left %d connections tracked as without a request", tracked)
	}
	if isPeerClosed(t, activePeer) {
		t.Fatal("Close closed a connection that sent a request")
	}
}

func TestServerClosesConnectionAcceptedAfterClose(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{Token: "tok"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Shutdown runs the hook in its own goroutine; the second run makes sure it has run before the
	// connection arrives, so only trackConn can close it.
	srv.closeNewConns()
	lateConn, latePeer := net.Pipe()
	cleanup.CloseAfter(t, lateConn)
	cleanup.CloseAfter(t, latePeer)

	srv.trackConn(lateConn, http.StateNew)

	if !isPeerClosed(t, latePeer) {
		t.Fatal("a connection accepted after Close stays open")
	}
}

package mcphttp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

func newTestServer(token string) *Server {
	return New(sdk.NewServer(&sdk.Implementation{Name: "test"}, nil), token)
}

type failingListener struct {
	err error
}

func (l *failingListener) Accept() (net.Conn, error) {
	return nil, l.err
}

func (l *failingListener) Close() error {
	return nil
}

func (l *failingListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 54321}
}

func TestServerOptionsAndRoutes(t *testing.T) {
	t.Parallel()

	streamOpts := newStreamableOptions()
	if streamOpts == nil || streamOpts.SessionTimeout != 30*time.Minute {
		t.Fatalf("unexpected streamable options: %+v", streamOpts)
	}

	srv := newTestServer("route-token")
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

func serveOnLoopback(t *testing.T, srv *Server) {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Serve(ln); err != nil {
		closeAndCheck(t, ln)
		t.Fatalf("Serve failed: %v", err)
	}
}

func TestServerSetTokenServeAndClose(t *testing.T) {
	t.Parallel()

	srv := newTestServer("tok-1")
	serveOnLoopback(t, srv)
	defer func() {
		if err := srv.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	url1 := srv.GetHTTPURL()
	if !strings.HasPrefix(url1, "http://127.0.0.1:") {
		t.Fatalf("expected loopback streamable HTTP URL, got %s", url1)
	}

	srv.SetToken("tok-2")
	if srv.GetToken() != "tok-2" || srv.GetHTTPURL() != url1 {
		t.Fatalf("expected token tok-2 and unchanged URL %s, got token=%s url=%s", url1, srv.GetToken(), srv.GetHTTPURL())
	}

	if err := srv.Close(t.Context()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if err := srv.Serve(&failingListener{err: net.ErrClosed}); err == nil {
		t.Fatal("expected Serve on closed server to fail")
	}
}

func TestServerServeErrorAndIdempotentClose(t *testing.T) {
	t.Parallel()

	srv := newTestServer("tok")

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

	srv := newTestServer("tok")
	serveOnLoopback(t, srv)
	addr := srv.actualAddr

	silentConn, err := new(net.Dialer).DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, silentConn)

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

	srv := newTestServer("tok")
	silentConn, silentPeer := net.Pipe()
	activeConn, activePeer := net.Pipe()
	for _, c := range []net.Conn{silentConn, silentPeer, activeConn, activePeer} {
		t.Cleanup(func() { closeAndCheck(t, c) })
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

	srv := newTestServer("tok")
	if err := srv.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Shutdown runs the hook in its own goroutine; the second run makes sure it has run before the
	// connection arrives, so only trackConn can close it.
	srv.closeNewConns()
	lateConn, latePeer := net.Pipe()
	t.Cleanup(func() { closeAndCheck(t, lateConn) })
	t.Cleanup(func() { closeAndCheck(t, latePeer) })

	srv.trackConn(lateConn, http.StateNew)

	if !isPeerClosed(t, latePeer) {
		t.Fatal("a connection accepted after Close stays open")
	}
}

// gatedDeadlineContext holds the first Deadline call until release is closed. Close calls Deadline
// after it has marked the server closed and before it calls http.Server.Shutdown.
type gatedDeadlineContext struct {
	context.Context
	once    sync.Once
	called  chan struct{}
	release chan struct{}
}

func (c *gatedDeadlineContext) Deadline() (time.Time, bool) {
	c.once.Do(func() { close(c.called) })
	<-c.release
	return c.Context.Deadline()
}

func waitSignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal(msg)
	}
}

// net/http has read the request and stored StateActive, and the ConnState hook for it runs only
// after Close has marked the server closed. net/http drops the request when its hook returns after
// Shutdown has started, so the test lets Shutdown start only once the handler runs.
func TestServerCloseAnswersRequestReadBeforeClose(t *testing.T) {
	t.Parallel()

	srv := newTestServer("tok")
	requestRead := make(chan struct{})
	releaseHook := make(chan struct{})
	trackConn := srv.httpServer.ConnState
	srv.httpServer.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateActive {
			close(requestRead)
			<-releaseHook
		}
		trackConn(conn, state)
	}
	handlerStarted := make(chan struct{})
	handler := srv.httpServer.Handler
	srv.httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerStarted)
		handler.ServeHTTP(w, r)
	})
	serveOnLoopback(t, srv)

	conn, err := new(net.Dialer).DialContext(t.Context(), "tcp", srv.actualAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, conn)
	if _, err := conn.Write([]byte("GET /mcp HTTP/1.1\r\nHost: numen.test\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	waitSignal(t, requestRead, "the server did not read the request")

	ctx := &gatedDeadlineContext{Context: t.Context(), called: make(chan struct{}), release: make(chan struct{})}
	closeErr := make(chan error, 1)
	go func() { closeErr <- srv.Close(ctx) }()
	waitSignal(t, ctx.called, "Close did not read the deadline of its context")
	close(releaseHook)
	waitSignal(t, handlerStarted, "the handler did not start")
	close(ctx.release)

	if err := conn.SetReadDeadline(clock.System{}.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("a request read before Close got no response: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if err := <-closeErr; err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func closeAndCheck(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}

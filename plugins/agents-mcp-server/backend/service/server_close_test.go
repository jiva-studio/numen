package service

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
)

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

	srv, err := NewServer(Config{Token: "tok"}, nil)
	if err != nil {
		t.Fatal(err)
	}
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
	defer cleanup.Close(t, conn)
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

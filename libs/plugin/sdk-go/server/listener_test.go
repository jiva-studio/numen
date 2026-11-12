package server

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strconv"
	"testing"
	"time"
)

var readyListenerPattern = regexp.MustCompile(`NUMEN_READY port=\d+[^\n]* listener\.mcp=(\d+)[ \n]`)

func waitForListenerPort(t *testing.T, stdout *safeBuffer) int {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		if match := readyListenerPattern.FindStringSubmatch(stdout.String()); match != nil {
			port, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("listener port %q: %v", match[1], err)
			}
			return port
		}
		select {
		case <-deadline:
			t.Fatalf("no ready line with listener.mcp, stdout %q", stdout.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func serveInBackground(s *Server) <-chan error {
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	return serveErrCh
}

func stopServer(t *testing.T, s *Server, serveErrCh <-chan error) {
	t.Helper()
	s.Lifecycle().TriggerShutdown()
	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Errorf("Serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Serve did not return after shutdown")
	}
}

func dialLoopback(ctx context.Context, port int) (net.Conn, error) {
	dialer := net.Dialer{Timeout: time.Second}
	return dialer.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
}

func closeConn(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Errorf("close: %v", err)
	}
}

func getListenerPort(t *testing.T, ln net.Listener) int {
	t.Helper()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", ln.Addr())
	}
	return addr.Port
}

func TestServeReportsTheListenerPortInTheReadyLine(t *testing.T) {
	var stdout safeBuffer
	s := New(WithPort(0), WithToken("listener-token"), WithStdout(&stdout), WithStdin(nil), WithSignals())
	listener, err := s.OpenListener("mcp")
	if err != nil {
		t.Fatalf("OpenListener: %v", err)
	}
	accepted := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			err = conn.Close()
		}
		accepted <- err
	}()
	serveErrCh := serveInBackground(s)
	t.Cleanup(func() { stopServer(t, s, serveErrCh) })

	port := waitForListenerPort(t, &stdout)

	if want := getListenerPort(t, listener); port != want {
		t.Errorf("ready line reports listener.mcp=%d, the listener is bound on %d", port, want)
	}
	conn, err := dialLoopback(t.Context(), port)
	if err != nil {
		t.Fatalf("dial the listener port: %v", err)
	}
	defer closeConn(t, conn)
	select {
	case err := <-accepted:
		if err != nil {
			t.Errorf("accept on the listener: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the listener accepted no connection")
	}
}

func TestOpenListenerBindsLoopback(t *testing.T) {
	s := New(WithPort(0), WithToken("listener-token"), WithStdout(&safeBuffer{}), WithStdin(nil), WithSignals())
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close the server: %v", err)
		}
	})

	listener, err := s.OpenListener("mcp")
	if err != nil {
		t.Fatalf("OpenListener: %v", err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close the listener: %v", err)
		}
	})

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !addr.IP.IsLoopback() || addr.Port == 0 {
		t.Errorf("listener address = %v, want 127.0.0.1 with an allocated port", listener.Addr())
	}
}

func TestServeClosesTheListenersItOpenedWhenItReturns(t *testing.T) {
	var stdout safeBuffer
	s := New(WithPort(0), WithToken("listener-token"), WithStdout(&stdout), WithStdin(nil), WithSignals())
	if _, err := s.OpenListener("mcp"); err != nil {
		t.Fatalf("OpenListener: %v", err)
	}
	serveErrCh := serveInBackground(s)
	port := waitForListenerPort(t, &stdout)

	stopServer(t, s, serveErrCh)

	conn, err := dialLoopback(t.Context(), port)
	if err == nil {
		closeConn(t, conn)
		t.Errorf("the listener port %d accepts connections after Serve returned", port)
	}
}

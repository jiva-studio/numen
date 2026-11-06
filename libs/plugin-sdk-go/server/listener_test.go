package server

import (
	"net"
	"regexp"
	"strconv"
	"testing"
	"time"
)

var readyListenerPattern = regexp.MustCompile(`NUMEN_READY port=\d+[^\n]* listener\.mcp=(\d+)[ \n]`)

// waitForListenerPort returns the port the ready line on stdout reports for the listener mcp.
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
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	t.Cleanup(func() {
		s.Lifecycle().TriggerShutdown()
		<-serveErrCh
	})

	port := waitForListenerPort(t, &stdout)

	if want := listener.Addr().(*net.TCPAddr).Port; port != want {
		t.Errorf("ready line reports listener.mcp=%d, the listener is bound on %d", port, want)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatalf("dial the listener port: %v", err)
	}
	defer func() { _ = conn.Close() }()
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
	t.Cleanup(func() { _ = s.Close() })

	listener, err := s.OpenListener("mcp")
	if err != nil {
		t.Fatalf("OpenListener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

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
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	port := waitForListenerPort(t, &stdout)

	s.Lifecycle().TriggerShutdown()
	select {
	case <-serveErrCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after shutdown")
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err == nil {
		_ = conn.Close()
		t.Errorf("the listener port %d accepts connections after Serve returned", port)
	}
}

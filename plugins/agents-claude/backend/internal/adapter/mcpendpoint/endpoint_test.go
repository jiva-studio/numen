package mcpendpoint

import (
	"net"
	"net/url"
	"testing"
	"time"
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

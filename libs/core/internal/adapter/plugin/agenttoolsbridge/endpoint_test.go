package agenttoolsbridge

import (
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

func dialEndpoint(t *testing.T, served *endpoint) net.Conn {
	t.Helper()
	address, err := url.Parse(served.url)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", address.Host)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, conn)
	return conn
}

func waitFor(t *testing.T, isReady func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if isReady() {
			return
		}
	}
	t.Fatal("the endpoint did not reach the expected state")
}

func TestCloseDoesNotWaitForAConnectionThatSentNoRequest(t *testing.T) {
	served, err := serveEndpoint(t.Context(), nil, callScope{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	dialEndpoint(t, served)
	waitFor(t, func() bool {
		served.statesMu.Lock()
		defer served.statesMu.Unlock()
		return len(served.states) > 0
	})

	started := time.Now()
	if err := served.close(t.Context()); err != nil {
		t.Errorf("close reported %v for a connection with no request", err)
	}
	if took := time.Since(started); took >= shutdownLimit/2 {
		t.Errorf("close took %v", took)
	}
}

func TestCloseReportsAnInterruptedRequest(t *testing.T) {
	served, err := serveEndpoint(t.Context(), nil, callScope{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	served.limit = 100 * time.Millisecond
	conn := dialEndpoint(t, served)
	if _, err := conn.Write([]byte("POST /mcp HTTP/1.1\r\nHost: stand-in\r\nContent-Length: 100\r\n\r\n{")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return served.getActiveCount() > 0 })

	if err := served.close(t.Context()); err == nil {
		t.Error("close reported nothing for a request it cut")
	}
}

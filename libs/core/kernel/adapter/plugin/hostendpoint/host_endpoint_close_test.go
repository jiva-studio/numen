package hostendpoint

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

// A client transport keeps a connection it dialed for a call that another connection served. Such a
// connection has no request in flight, and Close does not wait for it.
func TestCloseReturnsWhileAConnectionHasNoRequest(t *testing.T) {
	endpoint, err := Open(t.Context(), Address, newCallers().asCallers())
	if err != nil {
		t.Fatal(err)
	}
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", endpoint.GetPort()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Log(err)
		}
	})

	closeResult := make(chan error, 1)
	go func() { closeResult <- endpoint.Close(context.WithoutCancel(t.Context())) }()
	select {
	case err := <-closeResult:
		if err != nil {
			t.Errorf("Close with a connection with no request in flight returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close with a connection with no request in flight did not return")
	}
}

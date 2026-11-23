package synctest

import (
	"context"
	"net"
	"sync"
	"testing"
)

// Relay is a loopback TCP relay to a server. Pause makes every connection open at that moment swallow what it
// carries, as a half-open connection does; a connection opened later is carried as before.
type Relay struct {
	URL string

	mu     sync.Mutex
	conns  []*relayConn
	listen net.Listener
}

type relayConn struct {
	isFrozen bool
	mu       sync.Mutex
}

// NewRelay starts a relay to the host of the address (as http://host:port) and stops it when the test ends.
func NewRelay(t *testing.T, address string) *Relay {
	t.Helper()
	target := address[len("http://"):]
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	r := &Relay{URL: "http://" + listener.Addr().String(), listen: listener}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Error(err)
		}
	})
	go r.accept(target)
	return r
}

// Pause makes every open connection discard the bytes it receives.
func (r *Relay) Pause() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, conn := range r.conns {
		conn.mu.Lock()
		conn.isFrozen = true
		conn.mu.Unlock()
	}
}

func (r *Relay) accept(target string) {
	for {
		client, err := r.listen.Accept()
		if err != nil {
			return
		}
		upstream, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", target)
		if err != nil {
			closeQuietly(client)
			continue
		}
		state := &relayConn{}
		r.mu.Lock()
		r.conns = append(r.conns, state)
		r.mu.Unlock()
		go state.copy(upstream, client)
		go state.copy(client, upstream)
	}
}

func (c *relayConn) copy(to, from net.Conn) {
	defer closeQuietly(to)
	defer closeQuietly(from)
	buffer := make([]byte, 4096)
	for {
		n, err := from.Read(buffer)
		c.mu.Lock()
		isFrozen := c.isFrozen
		c.mu.Unlock()
		if n > 0 && !isFrozen {
			if _, writeErr := to.Write(buffer[:n]); writeErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// closeQuietly closes a connection of the relay whose peer may have closed it first; the error of a second
// close says nothing, as the copy ends either way.
func closeQuietly(conn net.Conn) {
	if err := conn.Close(); err != nil {
		return
	}
}

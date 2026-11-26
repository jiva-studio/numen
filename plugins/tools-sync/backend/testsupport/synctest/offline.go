package synctest

import (
	"errors"
	"net/http"
	"sync/atomic"
)

var errOffline = errors.New("this computer is offline")

// link is the connection of one computer to the network: its requests fail while it is off.
// A request for origin goes to dial: the public address of a server stays the address the computer dials, and the proxy of the test answers.
type link struct {
	isOffline    atomic.Bool
	origin, dial string
}

func (l *link) RoundTrip(r *http.Request) (*http.Response, error) {
	if l.isOffline.Load() {
		return nil, errOffline
	}
	if l.dial != "" && r.URL.Host == l.origin {
		r = r.Clone(r.Context())
		r.URL.Host = l.dial
	}
	return http.DefaultTransport.RoundTrip(r)
}

// GoOffline makes every request of this computer to the server fail until GoOnline.
func (c *Computer) GoOffline() { c.network.isOffline.Store(true) }

// GoOnline lets the requests of this computer reach the server again.
func (c *Computer) GoOnline() { c.network.isOffline.Store(false) }

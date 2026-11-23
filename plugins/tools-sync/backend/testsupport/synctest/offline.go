package synctest

import (
	"errors"
	"net/http"
	"sync/atomic"
)

var errOffline = errors.New("this computer is offline")

// link is the connection of one computer to the network: its requests fail while it is off.
type link struct {
	isOffline atomic.Bool
}

func (l *link) RoundTrip(r *http.Request) (*http.Response, error) {
	if l.isOffline.Load() {
		return nil, errOffline
	}
	return http.DefaultTransport.RoundTrip(r)
}

// GoOffline makes every request of this computer to the server fail until GoOnline.
func (c *Computer) GoOffline() { c.network.isOffline.Store(true) }

// GoOnline lets the requests of this computer reach the server again.
func (c *Computer) GoOnline() { c.network.isOffline.Store(false) }

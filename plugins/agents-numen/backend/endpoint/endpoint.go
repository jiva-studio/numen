// Package endpoint decides which addresses may receive a credential: https anywhere, and http
// only to this machine.
package endpoint

import (
	"errors"
	"net"
	"net/http"
	"net/url"
)

// ErrDowngrade is a redirect that would send a credential over plain http.
var ErrDowngrade = errors.New("a redirect would send a key over plain http")

const maxRedirects = 10

// IsLocalHost is whether host is localhost or a loopback address.
func IsLocalHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// IsPrivate is whether a credential may be sent to the address.
func IsPrivate(address *url.URL) bool {
	if address.Scheme == "https" {
		return true
	}
	if address.Scheme != "http" {
		return false
	}
	return IsLocalHost(address.Hostname())
}

// RefuseDowngrade is the CheckRedirect of a client that sends a credential: it stops a redirect
// to an address that is not private and one that follows ten others.
func RefuseDowngrade(req *http.Request, via []*http.Request) error {
	if !IsPrivate(req.URL) {
		return ErrDowngrade
	}
	if len(via) >= maxRedirects {
		return errors.New("stopped after ten redirects")
	}
	return nil
}

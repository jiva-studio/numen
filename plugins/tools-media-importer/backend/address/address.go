// Package address reads the web address a link holds.
package address

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

// A URL is a web address in the one form two spellings of one resource share.
// What is at it is the fetcher's to say.
type URL string

// ErrNotAURL is anything a link cannot point at. Only `http` and `https`
// are fetched, and only away from this machine: neither a scheme reaching a
// file nor a host that is this machine is an address a link may carry.
var ErrNotAURL = errors.New("not a web address")

// Parse reads a pasted address.
//
// The scheme and the host are lowercased, a default port and a fragment are
// dropped, and the parameters a site adds to say where a visitor came from are
// dropped with them; what is left is written in one order.
func Parse(raw string) (URL, error) {
	address, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", ErrNotAURL
	}
	address.Scheme = strings.ToLower(address.Scheme)
	if address.Scheme != "http" && address.Scheme != "https" {
		return "", ErrNotAURL
	}
	address.Host = strings.ToLower(address.Host)
	if address.Hostname() == "" || isLocalHost(address.Hostname()) {
		return "", ErrNotAURL
	}
	if port := address.Port(); port == "80" && address.Scheme == "http" ||
		port == "443" && address.Scheme == "https" {
		address.Host = address.Hostname()
	}
	address.Fragment, address.RawFragment = "", ""
	address.User = nil
	address.RawQuery = removeTrackingParams(address.Query()).Encode()
	return URL(address.String()), nil
}

// isLocalHost reports whether a host is this machine. A file points at a
// location a browser would visit, and services listening on this machine are
// not that: an index, a window's own socket and other local processes are not
// valid pasted addresses.
func isLocalHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	address, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return false
	}
	return address.IsLoopback() || address.IsUnspecified()
}

// trackingParams are the parameters that say where a visitor came from, beside every
// name beginning `utm_`. They are the same page whatever they hold, and a
// person who pastes one link from two places holds one file about it.
var trackingParams = map[string]bool{
	"fbclid":  true,
	"gclid":   true,
	"mc_cid":  true,
	"mc_eid":  true,
	"si":      true,
	"ref_src": true,
	"ref_url": true,
}

// removeTrackingParams is the query without them.
func removeTrackingParams(query url.Values) url.Values {
	for name := range query {
		folded := strings.ToLower(name)
		if trackingParams[folded] || strings.HasPrefix(folded, "utm_") {
			delete(query, name)
		}
	}
	return query
}

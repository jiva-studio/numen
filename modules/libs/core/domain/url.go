package domain

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

// A URL is where a `.url` file points, in the one form two spellings of one
// resource share. What was fetched from it is kept under a name made from this
// form, so two files that pasted the same address in different words hold one
// text between them.
//
// What is at it — a video, a page, anything else — is not this type's to say.
// Whoever fetches an address answers for the sites it knows.
type URL string

// ErrNotAURL is anything a `.url` file cannot point at. Only `http` and `https`
// are fetched, and only away from this machine: neither a scheme reaching a
// file nor a host that is this machine is an address a file may carry.
var ErrNotAURL = errors.New("not a web address")

// ParseURL reads what a person pasted.
//
// The scheme and the host are lowercased, a default port and a fragment are
// dropped, and the parameters a site adds to say where a visitor came from are
// dropped with them; what is left is written in one order.
func ParseURL(raw string) (URL, error) {
	address, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", ErrNotAURL
	}
	address.Scheme = strings.ToLower(address.Scheme)
	if address.Scheme != "http" && address.Scheme != "https" {
		return "", ErrNotAURL
	}
	address.Host = strings.ToLower(address.Host)
	if address.Hostname() == "" || thisMachine(address.Hostname()) {
		return "", ErrNotAURL
	}
	if port := address.Port(); port == "80" && address.Scheme == "http" ||
		port == "443" && address.Scheme == "https" {
		address.Host = address.Hostname()
	}
	address.Fragment, address.RawFragment = "", ""
	address.User = nil
	address.RawQuery = kept(address.Query()).Encode()
	return URL(address.String()), nil
}

// thisMachine says whether a host is this machine. A file points somewhere a
// browser would go, and what listens on this machine is not that: an index, a
// window's own socket and whatever else is running answer nobody's paste.
func thisMachine(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	address, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return false
	}
	return address.IsLoopback() || address.IsUnspecified()
}

// campaign are the parameters that say where a visitor came from, beside every
// name beginning `utm_`. They are the same page whatever they hold, and a
// person who pastes one link from two places holds one file about it.
var campaign = map[string]bool{
	"fbclid":  true,
	"gclid":   true,
	"mc_cid":  true,
	"mc_eid":  true,
	"si":      true,
	"ref_src": true,
	"ref_url": true,
}

// kept is the query without them.
func kept(query url.Values) url.Values {
	for name := range query {
		folded := strings.ToLower(name)
		if campaign[folded] || strings.HasPrefix(folded, "utm_") {
			delete(query, name)
		}
	}
	return query
}

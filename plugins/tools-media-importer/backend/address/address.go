// Package address reads the web address a link holds.
package address

import (
	"errors"
	"net/url"
	"strings"
)

// A URL is a web address in the one form two spellings of one resource share.
// What is at it is the fetcher's to say.
type URL string

// ErrNotAURL is an address with a scheme other than `http` and `https`, or
// with an empty host.
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
	// gomutants:disable-next-line STATEMENT_REMOVE reason="url.Parse already lowercases Scheme"
	address.Scheme = strings.ToLower(address.Scheme)
	if address.Scheme != "http" && address.Scheme != "https" {
		return "", ErrNotAURL
	}
	address.Host = strings.ToLower(address.Host)
	if address.Hostname() == "" {
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

// isTrackingParam reports whether the parameter says where a visitor came from, beside every name
// beginning `utm_`. Such a parameter is the same page whatever it holds, and a person who pastes
// one link from two places holds one file about it.
func isTrackingParam(key string) bool {
	switch key {
	case "fbclid", "gclid", "mc_cid", "mc_eid", "si", "ref_src", "ref_url":
		return true
	}
	return false
}

// removeTrackingParams is the query without them.
func removeTrackingParams(query url.Values) url.Values {
	for name := range query {
		folded := strings.ToLower(name)
		if isTrackingParam(folded) || strings.HasPrefix(folded, "utm_") {
			delete(query, name)
		}
	}
	return query
}

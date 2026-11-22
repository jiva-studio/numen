package syncprotocol

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var errBadAddress = errors.New("not an address of a server")

// NormalizeAddress returns the scheme, host and port of an http or https address in lower case, without
// the port the scheme implies and without a trailing slash. An address with a user, a path, a query or a
// fragment is not one.
func NormalizeAddress(address string) (string, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %w", errBadAddress, address, err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	hasExtra := parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != ""
	if (scheme != "http" && scheme != "https") || host == "" || hasExtra {
		return "", fmt.Errorf("%w: %q", errBadAddress, address)
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	port := parsed.Port()
	if port == "" || (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		return scheme + "://" + host, nil
	}
	return scheme + "://" + host + ":" + port, nil
}

// IsSameAddress reports whether both are addresses of a server and name the same scheme, host and port.
func IsSameAddress(a, b string) bool {
	left, err := NormalizeAddress(a)
	if err != nil {
		return false
	}
	right, err := NormalizeAddress(b)
	return err == nil && left == right
}

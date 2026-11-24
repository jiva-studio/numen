package syncprotocol

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

// ErrInsecureAddress is the refusal of a server address that is neither https nor http on a loopback host.
var ErrInsecureAddress = errors.New("the sync server address must be https, or http on this computer")

// IsLoopbackHost reports whether host is localhost or a loopback IP address.
func IsLoopbackHost(host string) bool {
	host = strings.ToLower(strings.Trim(host, "[]"))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

// CheckServerAddress returns nil for an https address and for an http address on a loopback host. A bearer, a
// signature and a blob never go over plain http to any other host.
func CheckServerAddress(address string) error {
	normal, err := NormalizeAddress(address)
	if err != nil {
		return err
	}
	parsed, err := url.Parse(normal)
	if err != nil {
		return err
	}
	if parsed.Scheme == "https" || IsLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return ErrInsecureAddress
}

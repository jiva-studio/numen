// Package address decides which addresses a request may go to. A request goes only to an http or https
// address with a host and no credentials. A key goes over http only to a loopback address.
// Link-local, unspecified, multicast and cloud metadata addresses are refused.
package address

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

// SchemeHTTP is the scheme of an address that carries no encryption.
const SchemeHTTP = "http"

// Error is an address a request may not go to, with the reason.
type Error struct{ Reason string }

// Error returns the reason.
func (e *Error) Error() string { return e.Reason }

// IsError reports whether err is, or wraps, an address that is refused.
func IsError(err error) bool {
	var invalid *Error
	return errors.As(err, &invalid)
}

// Parse reads the address of a service. hasKey is whether a key travels with the request.
func Parse(address string, hasKey bool) (*url.URL, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, &Error{"the address is not a URL"}
	}
	if parsed.Scheme != SchemeHTTP && parsed.Scheme != "https" {
		return nil, &Error{"the address must start with http:// or https://"}
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" {
		return nil, &Error{"the address has no host"}
	}
	if parsed.User != nil {
		return nil, &Error{"the address holds a user name or a password"}
	}
	if isForbiddenHost(host) {
		return nil, &Error{"the address names a host that is not allowed"}
	}
	if parsed.Scheme == SchemeHTTP && hasKey && !isLoopbackHost(host) {
		return nil, &Error{"a key is sent over http only to this computer"}
	}
	return parsed, nil
}

// Check reports why Parse refuses the address, or nil.
func Check(address string, hasKey bool) error {
	_, err := Parse(address, hasKey)
	return err
}

// CheckResolved refuses an IP address a host name resolved to when it is not allowed, or, where a key
// travels over http, when it is not a loopback address.
func CheckResolved(ip netip.Addr, isLoopbackRequired bool) error {
	ip = ip.Unmap().WithZone("")
	if isForbiddenAddress(ip) {
		return &Error{"the address resolves to a host that is not allowed"}
	}
	if isLoopbackRequired && !ip.IsLoopback() {
		return &Error{"a key is sent over http only to this computer"}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Unmap().IsLoopback()
}

func isForbiddenHost(host string) bool {
	if host == "metadata" || host == "metadata.google.internal" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && isForbiddenAddress(ip.Unmap().WithZone(""))
}

func isForbiddenAddress(ip netip.Addr) bool {
	switch ip.String() {
	case "100.100.100.200", "fd00:ec2::254":
		return true
	default:
	}
	return ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

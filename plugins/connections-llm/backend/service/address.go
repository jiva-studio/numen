package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

type addressError struct{ reason string }

func (e *addressError) Error() string { return e.reason }

type loopbackOnlyKey struct{}

func parseAddress(address string, hasKey bool) (*url.URL, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, &addressError{"the address is not a URL"}
	}
	if parsed.Scheme != schemeHTTP && parsed.Scheme != "https" {
		return nil, &addressError{"the address must start with http:// or https://"}
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" {
		return nil, &addressError{"the address has no host"}
	}
	if parsed.User != nil {
		return nil, &addressError{"the address holds a user name or a password"}
	}
	if isForbiddenHost(host) {
		return nil, &addressError{"the address names a host that is not allowed"}
	}
	if parsed.Scheme == schemeHTTP && hasKey && !isLoopbackHost(host) {
		return nil, &addressError{"a key is sent over http only to this computer"}
	}
	return parsed, nil
}

func checkAddress(address string, hasKey bool) error {
	_, err := parseAddress(address, hasKey)
	return err
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

func checkResolved(ip netip.Addr, isLoopbackRequired bool) error {
	ip = ip.Unmap().WithZone("")
	if isForbiddenAddress(ip) {
		return &addressError{"the address resolves to a host that is not allowed"}
	}
	if isLoopbackRequired && !ip.IsLoopback() {
		return &addressError{"a key is sent over http only to this computer"}
	}
	return nil
}

func controlDial(ctx context.Context, _, address string, _ syscall.RawConn) error {
	target, err := netip.ParseAddrPort(address)
	if err != nil {
		return &addressError{fmt.Sprintf("the dialed address %q is not an IP address", address)}
	}
	isLoopbackRequired, ok := ctx.Value(loopbackOnlyKey{}).(bool)
	return checkResolved(target.Addr(), ok && isLoopbackRequired)
}

// clients holds one client for requests that may go to any allowed address and one for requests that
// carry a key over http, so a connection made for the first never serves the second.
type clients struct {
	general, loopbackOnly *http.Client
}

func newClients() clients {
	return clients{general: newClient(), loopbackOnly: newClient()}
}

func (c clients) close() {
	c.general.CloseIdleConnections()
	c.loopbackOnly.CloseIdleConnections()
}

func (c clients) get(isLoopbackRequired bool) *http.Client {
	if isLoopbackRequired {
		return c.loopbackOnly
	}
	return c.general
}

func newClient() *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second, ControlContext: controlDial}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         dialer.DialContext,
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 30 * time.Second,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 8,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func isAddressError(err error) bool {
	var invalid *addressError
	return errors.As(err, &invalid)
}

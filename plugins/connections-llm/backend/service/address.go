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

type refusedError struct{ reason string }

func (e *refusedError) Error() string { return e.reason }

type loopbackOnlyKey struct{}

// parseAddress accepts an http or https address with a host and no credentials. A key goes over
// http only to a loopback host, and link-local, unspecified, multicast and cloud metadata hosts are
// refused. The host name is resolved at dial time, where checkResolved looks at the result.
func parseAddress(address string, hasKey bool) (*url.URL, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, &refusedError{"the address is not a URL"}
	}
	if parsed.Scheme != schemeHTTP && parsed.Scheme != "https" {
		return nil, &refusedError{"the address must start with http:// or https://"}
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" {
		return nil, &refusedError{"the address has no host"}
	}
	if parsed.User != nil {
		return nil, &refusedError{"the address holds a user name or a password"}
	}
	if isForbiddenHost(host) {
		return nil, &refusedError{"the address names a host that is not allowed"}
	}
	if parsed.Scheme == schemeHTTP && hasKey && !isLoopbackHost(host) {
		return nil, &refusedError{"a key is sent over http only to this computer"}
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

// checkResolved refuses a resolved address a key must not reach: forbidden addresses always, and
// every address but a loopback one when the key travels over http.
func checkResolved(ip netip.Addr, isLoopbackRequired bool) error {
	ip = ip.Unmap().WithZone("")
	if isForbiddenAddress(ip) {
		return &refusedError{"the address resolves to a host that is not allowed"}
	}
	if isLoopbackRequired && !ip.IsLoopback() {
		return &refusedError{"a key is sent over http only to this computer"}
	}
	return nil
}

func controlDial(ctx context.Context, _, address string, _ syscall.RawConn) error {
	target, err := netip.ParseAddrPort(address)
	if err != nil {
		return &refusedError{fmt.Sprintf("the dialed address %q is not an IP address", address)}
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

func isRefused(err error) bool {
	var refused *refusedError
	return errors.As(err, &refused)
}

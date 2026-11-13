package openaiapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"

	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
)

type loopbackOnlyKey struct{}

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

func controlDial(ctx context.Context, _, target string, _ syscall.RawConn) error {
	dialed, err := netip.ParseAddrPort(target)
	if err != nil {
		return &address.Error{Reason: fmt.Sprintf("the dialed address %q is not an IP address", target)}
	}
	isLoopbackRequired, ok := ctx.Value(loopbackOnlyKey{}).(bool)
	return address.CheckResolved(dialed.Addr(), ok && isLoopbackRequired)
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

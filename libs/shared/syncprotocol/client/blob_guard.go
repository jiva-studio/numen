package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

var errBlobAddress = errors.New("the sync server sent a storage address that Numen does not use")

func getCarrierGradeNAT() netip.Prefix { return netip.MustParsePrefix("100.64.0.0/10") }

// NewBlobHTTPClient returns the client that moves blobs over the presigned addresses of the server at
// serverAddress. It sends to https addresses only, and to http only on a loopback host when the server is
// on one; timeout is the time a transfer may stand still, not its total time; it never connects to a loopback, private, link-local or carrier-grade address unless the server
// is on a loopback host; and it follows no redirect. The dial check runs on the resolved address, so a name
// that resolves to such an address is refused. A request through a proxy is checked by its address only.
func NewBlobHTTPClient(serverAddress string, timeout time.Duration) *http.Client {
	isLocal := isLoopbackAddress(serverAddress)
	dialer := &net.Dialer{Timeout: timeout, Control: func(_, address string, _ syscall.RawConn) error {
		return checkDialAddress(address, isLocal)
	}}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}
	guarded := guardTransport{next: transport, isLocal: isLocal}
	return &http.Client{Transport: progressTransport{next: guarded, idle: timeout}, CheckRedirect: refuseRedirect}
}

type guardTransport struct {
	next    http.RoundTripper
	isLocal bool
}

func (g guardTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := checkBlobAddress(request.URL, g.isLocal); err != nil {
		return nil, err
	}
	return g.next.RoundTrip(request)
}

func isLoopbackAddress(address string) bool {
	parsed, err := url.Parse(address)
	return err == nil && syncprotocol.IsLoopbackHost(parsed.Hostname())
}

func checkBlobAddress(address *url.URL, isLocal bool) error {
	host := address.Hostname()
	if host == "" || (address.Scheme != "https" && !(address.Scheme == "http" && isLocal && syncprotocol.IsLoopbackHost(host))) {
		return fmt.Errorf("%w: %q is not https", errBlobAddress, address.Scheme)
	}
	if ip, err := netip.ParseAddr(host); err == nil && !isPublicAddress(ip, isLocal) {
		return fmt.Errorf("%w: the host %q is not a public address", errBlobAddress, host)
	}
	return nil
}

func checkDialAddress(address string, isLocal bool) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: %w", errBlobAddress, err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("%w: %w", errBlobAddress, err)
	}
	if !isPublicAddress(ip, isLocal) {
		return fmt.Errorf("%w: %q is not a public address", errBlobAddress, host)
	}
	return nil
}

func isPublicAddress(ip netip.Addr, isLocal bool) bool {
	ip = ip.Unmap()
	if ip.IsLoopback() {
		return isLocal
	}
	return !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() &&
		!ip.IsUnspecified() && !getCarrierGradeNAT().Contains(ip)
}

func refuseRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func withoutRedirects(httpClient *http.Client) *http.Client {
	copied := *httpClient
	copied.CheckRedirect = refuseRedirect
	return &copied
}

func redactURL(err error) error {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}

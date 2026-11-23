package syncprotocol

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

// PaymentProviderDomain is the domain of the payment provider; a payment page is served from it or from a
// subdomain of it.
const PaymentProviderDomain = "creem.io"

var errPaymentAddress = errors.New("the sync server sent a payment address that Numen does not open")

// CheckPaymentAddress returns nil for an address a browser may open to pay: https (or http on a loopback
// host) on the payment provider, or on the family of the sync server. A server on a loopback host is a
// development server and may send any https address. The error names the host and never the path or query.
func CheckPaymentAddress(serverAddress, address string) error {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("%w: it is not an address", errPaymentAddress)
	}
	host := strings.ToLower(parsed.Hostname())
	isLoopback := IsLoopbackHost(host)
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback) {
		return fmt.Errorf("%w: %q is not https", errPaymentAddress, parsed.Scheme)
	}
	if isLoopback || isPaymentHost(serverAddress, host) {
		return nil
	}
	return fmt.Errorf("%w: the host %q", errPaymentAddress, host)
}

func isPaymentHost(serverAddress, host string) bool {
	if isInDomain(host, PaymentProviderDomain) {
		return true
	}
	server, err := url.Parse(serverAddress)
	if err != nil || server.Hostname() == "" {
		return false
	}
	serverHost := strings.ToLower(server.Hostname())
	if IsLoopbackHost(serverHost) {
		return true
	}
	if _, err := netip.ParseAddr(serverHost); err == nil {
		return host == serverHost
	}
	return isInDomain(host, getDomainFamily(serverHost))
}

func getDomainFamily(host string) string {
	labels := strings.Split(host, ".")
	if len(labels) <= 2 {
		return host
	}
	return strings.Join(labels[len(labels)-2:], ".")
}

func isInDomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

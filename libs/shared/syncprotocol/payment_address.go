package syncprotocol

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// PaymentProviderDomain is the domain of the payment provider; a payment page is served from it or from a
// subdomain of it.
const PaymentProviderDomain = "creem.io"

var errPaymentAddress = errors.New("the sync server sent a payment address that Numen does not open")

// CheckPaymentAddress returns nil for an address a browser may open to pay: https on the payment provider, or on
// the family of the sync server, which is the registrable domain of its host under the public suffix list. A server
// on a loopback host is a development server: it may send any https address and an http address on a loopback host.
// No other server may send a loopback address. The error names the host and never the path or query.
func CheckPaymentAddress(serverAddress, address string) error {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("%w: it is not an address", errPaymentAddress)
	}
	host := strings.ToLower(parsed.Hostname())
	serverHost := getServerHost(serverAddress)
	isDevelopment := serverHost != "" && IsLoopbackHost(serverHost)
	isLoopback := IsLoopbackHost(host)
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback && isDevelopment) {
		return fmt.Errorf("%w: %q is not https", errPaymentAddress, parsed.Scheme)
	}
	if isDevelopment || (!isLoopback && isPaymentHost(serverHost, host)) {
		return nil
	}
	return fmt.Errorf("%w: the host %q", errPaymentAddress, host)
}

func getServerHost(serverAddress string) string {
	server, err := url.Parse(serverAddress)
	if err != nil {
		return ""
	}
	return strings.ToLower(server.Hostname())
}

func isPaymentHost(serverHost, host string) bool {
	if isInDomain(host, PaymentProviderDomain) {
		return true
	}
	if serverHost == "" {
		return false
	}
	if _, err := netip.ParseAddr(serverHost); err == nil {
		return host == serverHost
	}
	return isInDomain(host, getDomainFamily(serverHost))
}

func getDomainFamily(host string) string {
	family, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return family
}

func isInDomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

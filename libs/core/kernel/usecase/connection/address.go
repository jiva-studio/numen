package connection

import (
	"net/netip"
	"net/url"
	"strings"
)

// CheckAddress accepts an http or https address with a host and no credentials. Over http a key goes
// only to a loopback address. Link-local, unspecified and cloud metadata addresses are refused. A private network is accepted. A host name other than localhost is not
// resolved here, so the provider checks the address it resolves to.
func CheckAddress(address string, hasKey bool) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return newInvalidError("the address is not a URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return newInvalidError("the address must start with http:// or https://")
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" {
		return newInvalidError("the address has no host")
	}
	if parsed.User != nil {
		return newInvalidError("the address must not hold a user name or a password")
	}
	class := classifyHost(host)
	switch {
	case class == hostForbidden:
		return newInvalidError("the address names a host that is not allowed: %s", host)
	case parsed.Scheme == "http" && hasKey && class != hostLoopback:
		return newInvalidError("a key is sent over http only to this computer; use https")
	}
	return nil
}

type hostClass int

const (
	hostPublic hostClass = iota
	hostLoopback
	hostForbidden
)

func classifyHost(host string) hostClass {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return hostLoopback
	}
	if host == "metadata" || host == "metadata.google.internal" {
		return hostForbidden
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return hostPublic
	}
	ip = ip.Unmap().WithZone("")
	switch {
	case ip.IsLoopback():
		return hostLoopback
	case ip.IsUnspecified(), ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(), ip.IsMulticast(), isMetadataAddress(ip):
		return hostForbidden
	}
	return hostPublic
}

func isMetadataAddress(ip netip.Addr) bool {
	return ip == netip.MustParseAddr("100.100.100.200") || ip == netip.MustParseAddr("fd00:ec2::254")
}

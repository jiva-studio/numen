package syncprotocol_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

func TestCheckPaymentAddress_AcceptsOnlyThePaymentProviderAndTheFamilyOfTheServer(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		name, server, address string
		isAccepted            bool
	}{
		{"the provider", "https://sync.numen.md", "https://checkout.creem.io/pay/1", true},
		{"a subdomain of the provider", "https://sync.numen.md", "https://test-checkout.creem.io/pay/1", true},
		{"the server itself", "https://sync.numen.md", "https://sync.numen.md/portal/1", true},
		{"a sibling of the server", "https://sync.numen.md", "https://pay.numen.md/portal/1", true},
		{"a look-alike of the provider", "https://sync.numen.md", "https://creem.io.evil.example/pay/1", false},
		{"a provider suffix without a dot", "https://sync.numen.md", "https://evilcreem.io/pay/1", false},
		{"another site", "https://sync.numen.md", "https://numen-pay.attacker.example/pay/1", false},
		{"plain http on the provider", "https://sync.numen.md", "http://checkout.creem.io/pay/1", false},
		{"plain http on a remote host", "https://sync.numen.md", "http://sync.numen.md/pay/1", false},
		{"a development server sends its own page", "http://127.0.0.1:8080", "http://127.0.0.1:8080/checkout/1", true},
		{"a development server sends any https page", "http://127.0.0.1:8080", "https://pay.example/checkout/1", true},
		{"a loopback page from a remote server", "https://sync.numen.md", "http://127.0.0.1:8080/checkout/1", false},
		{"a localhost https page from a remote server", "https://sync.numen.md", "https://localhost/checkout/1", false},
		{"a sibling under a two-part suffix", "https://sync.example.co.uk", "https://pay.example.co.uk/portal/1", true},
		{"another site under the same two-part suffix", "https://sync.example.co.uk", "https://evil.co.uk/portal/1", false},
		{"another tenant of a private suffix", "https://sync.alice.github.io", "https://bob.github.io/portal/1", false},
		{"a file", "https://sync.numen.md", "file:///etc/passwd", false},
		{"no host", "https://sync.numen.md", "https:///checkout", false},
		{"empty", "https://sync.numen.md", "", false},
	} {
		if err := syncprotocol.CheckPaymentAddress(one.server, one.address); (err == nil) != one.isAccepted {
			t.Errorf("%s: CheckPaymentAddress(%q, %q) = %v, accepted should be %v", one.name, one.server, one.address, err, one.isAccepted)
		}
	}
}

func TestCheckPaymentAddress_NamesTheHostAndNeverThePathOrQuery(t *testing.T) {
	t.Parallel()
	err := syncprotocol.CheckPaymentAddress("https://sync.numen.md", "https://evil.example/pay/1?token=secret")

	if err == nil || !strings.Contains(err.Error(), "evil.example") || strings.Contains(err.Error(), "secret") {
		t.Errorf("error = %v, want the host and no query", err)
	}
}

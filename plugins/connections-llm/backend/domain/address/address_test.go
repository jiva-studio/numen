package address

import (
	"net/netip"
	"testing"
)

func TestAnAddressThatCannotCarryAKeyIsRefused(t *testing.T) {
	addresses := map[string]string{
		"not a URL":                       "::not a url",
		"a scheme other than http":        "ftp://example.com/v1",
		"no scheme":                       "example.com/v1",
		"no host":                         "https:///v1",
		"a user name":                     "https://user:pass@example.com/v1",
		"a link-local address":            "http://169.254.169.254/latest",
		"the metadata name":               "https://metadata.google.internal/v1",
		"the unspecified host":            "http://0.0.0.0:8080/v1",
		"plain http to a host":            "http://example.com/v1",
		"plain http to a private address": "http://192.168.1.20/v1",
	}
	for name, address := range addresses {
		err := Check(address, true)
		if err == nil || !IsError(err) {
			t.Errorf("%s: err = %v, want a refusal", name, err)
		}
	}
}

func TestAnAddressWithoutAKeyMayUsePlainHTTPOnAPrivateNetwork(t *testing.T) {
	for _, address := range []string{"http://192.168.1.20:11434/v1", "http://ollama.lan:11434/v1", "https://example.com/v1"} {
		if err := Check(address, false); err != nil {
			t.Errorf("%s: err = %v", address, err)
		}
	}
}

func TestAKeyMayGoOverPlainHTTPToTheLoopbackOnly(t *testing.T) {
	for _, address := range []string{"http://localhost:11434/v1", "http://127.0.0.1:8080", "http://[::1]:8080", "http://chat.localhost/v1"} {
		if err := Check(address, true); err != nil {
			t.Errorf("%s: err = %v", address, err)
		}
	}
}

func TestAResolvedAddressIsCheckedBeforeAConnectionIsMade(t *testing.T) {
	cases := []struct {
		address        string
		hasKey         bool
		isPlain        bool
		isAddressError bool
	}{
		{"127.0.0.1", true, true, false},
		{"::1", true, true, false},
		{"::ffff:127.0.0.1", true, true, false},
		{"93.184.216.34", true, false, false},
		{"93.184.216.34", true, true, true},
		{"93.184.216.34", false, true, false},
		{"10.0.0.8", false, true, false},
		{"169.254.169.254", false, false, true},
		{"fe80::1", false, false, true},
		{"0.0.0.0", false, false, true},
		{"::", false, false, true},
		{"224.0.0.1", false, false, true},
		{"100.100.100.200", false, false, true},
		{"fd00:ec2::254", false, false, true},
	}
	for _, one := range cases {
		err := CheckResolved(netip.MustParseAddr(one.address), one.hasKey && one.isPlain)
		if (err != nil) != one.isAddressError {
			t.Errorf("%s (key over http %v): err = %v, refused %v", one.address, one.hasKey && one.isPlain, err, one.isAddressError)
		}
	}
}

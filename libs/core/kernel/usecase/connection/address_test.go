package connection_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
)

func TestCheckAddressAcceptsWhatAConnectionMayReach(t *testing.T) {
	cases := []struct {
		name, address string
		hasKey        bool
	}{
		{"an https address", "https://api.example.com/v1", true},
		{"an http address with no key", "http://api.example.com/v1", false},
		{"http on localhost with a key", "http://localhost:11434/v1", true},
		{"http on a loopback address with a key", "http://127.0.0.1:8080", true},
		{"http on the IPv6 loopback with a key", "http://[::1]:8080", true},
		{"a private address", "http://192.168.1.20:11434/v1", false},
		{"a private address over https", "https://10.0.0.5", true},
		{"a public address literal", "https://8.8.8.8", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := connection.CheckAddress(c.address, c.hasKey); err != nil {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestCheckAddressRefusesWhatAConnectionMustNotReach(t *testing.T) {
	cases := []struct {
		name, address string
		hasKey        bool
	}{
		{"a scheme that is not http", "ftp://example.com", false},
		{"a file address", "file:///etc/passwd", false},
		{"a script address", "javascript:alert(1)", false},
		{"no scheme", "example.com/v1", false},
		{"no host", "https://", false},
		{"credentials in the address", "https://user:pass@example.com", false},
		{"http with a key on a public host", "http://api.example.com/v1", true},
		{"http with a key on a private address", "http://192.168.1.20", true},
		{"the metadata address of the clouds", "http://169.254.169.254/latest", false},
		{"a link-local address over https", "https://169.254.10.10", false},
		{"the IPv6 link-local range", "https://[fe80::1]", false},
		{"the metadata address of AWS over IPv6", "http://[fd00:ec2::254]", false},
		{"the metadata name of Google", "http://metadata.google.internal", false},
		{"the metadata address of Alibaba", "http://100.100.100.200", false},
		{"the unspecified address", "http://0.0.0.0:80", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := connection.CheckAddress(c.address, c.hasKey); !errors.Is(err, connection.ErrInvalidField) {
				t.Errorf("err = %v, want ErrInvalidField", err)
			}
		})
	}
}

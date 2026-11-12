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
		isPrivate     bool
	}{
		{"an https address", "https://api.example.com/v1", true, false},
		{"an http address with no key", "http://api.example.com/v1", false, false},
		{"http on localhost with a key", "http://localhost:11434/v1", true, false},
		{"http on a loopback address with a key", "http://127.0.0.1:8080", true, false},
		{"http on the IPv6 loopback with a key", "http://[::1]:8080", true, false},
		{"a private address when the person allows it", "http://192.168.1.20:11434/v1", false, true},
		{"a private address over https when the person allows it", "https://10.0.0.5", true, true},
		{"a public address literal", "https://8.8.8.8", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := connection.CheckAddress(c.address, c.hasKey, c.isPrivate); err != nil {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestCheckAddressRefusesWhatAConnectionMustNotReach(t *testing.T) {
	cases := []struct {
		name, address string
		hasKey        bool
		isPrivate     bool
	}{
		{"a scheme that is not http", "ftp://example.com", false, false},
		{"a file address", "file:///etc/passwd", false, false},
		{"a script address", "javascript:alert(1)", false, false},
		{"no scheme", "example.com/v1", false, false},
		{"no host", "https://", false, false},
		{"credentials in the address", "https://user:pass@example.com", false, false},
		{"http with a key on a public host", "http://api.example.com/v1", true, false},
		{"http with a key on a private address", "http://192.168.1.20", true, true},
		{"the metadata address of the clouds", "http://169.254.169.254/latest", false, true},
		{"a link-local address over https", "https://169.254.10.10", false, true},
		{"the IPv6 link-local range", "https://[fe80::1]", false, true},
		{"the metadata address of AWS over IPv6", "http://[fd00:ec2::254]", false, true},
		{"the metadata name of Google", "http://metadata.google.internal", false, true},
		{"the metadata address of Alibaba", "http://100.100.100.200", false, true},
		{"the unspecified address", "http://0.0.0.0:80", false, true},
		{"a private address when the person does not allow it", "http://10.0.0.5", false, false},
		{"a unique local address when the person does not allow it", "https://[fd12:3456::1]", false, false},
		{"an address of the shared range when the person does not allow it", "https://100.64.0.1", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := connection.CheckAddress(c.address, c.hasKey, c.isPrivate); !errors.Is(err, connection.ErrInvalidField) {
				t.Errorf("err = %v, want ErrInvalidField", err)
			}
		})
	}
}

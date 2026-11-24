package syncprotocol_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

func TestCheckServerAddress_AcceptsHTTPSAndLoopbackHTTPOnly(t *testing.T) {
	t.Parallel()
	for address, isAccepted := range map[string]bool{
		"https://sync.example.org":      true,
		"https://sync.example.org:8443": true,
		"http://localhost:8080":         true,
		"http://127.0.0.1:8080":         true,
		"http://[::1]:8080":             true,
		"http://sync.example.org":       false,
		"http://192.168.1.20:8080":      false,
		"http://localhost.example.org":  false,
		"ftp://sync.example.org":        false,
		"":                              false,
	} {
		if err := syncprotocol.CheckServerAddress(address); (err == nil) != isAccepted {
			t.Errorf("CheckServerAddress(%q) = %v, accepted should be %v", address, err, isAccepted)
		}
	}
}

func TestCheckServerAddress_NamesTheInsecureAddress(t *testing.T) {
	t.Parallel()
	if err := syncprotocol.CheckServerAddress("http://sync.example.org"); !errors.Is(err, syncprotocol.ErrInsecureAddress) {
		t.Fatalf("got %v, want ErrInsecureAddress", err)
	}
}

package epochtrust

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// DefaultServer is the address of the server of the build, the only one the embedded roots are for.
const DefaultServer = "https://sync.numen.md"

// EmbeddedRootHex is the public root key of the default server, Ed25519 in hex.
const EmbeddedRootHex = "9eecd55a6ae5b10e7ec349adac737207155fb5e8970a4b565590bd80d1b17dac"

func getEmbeddedRoots() []ed25519.PublicKey {
	key, err := hex.DecodeString(EmbeddedRootHex)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil
	}
	return []ed25519.PublicKey{key}
}

// GetRoots returns the roots that count for the server at address: the embedded one for the default server and
// only the root key of the plugin setting (hex) for any other server. A setting never adds a root for the default
// server and the embedded root never counts for another server.
func GetRoots(address, settingRoot string) ([]ed25519.PublicKey, error) {
	if isDefaultServer(address) {
		return getEmbeddedRoots(), nil
	}
	settingRoot = strings.TrimSpace(settingRoot)
	if settingRoot == "" {
		return nil, nil
	}
	key, err := hex.DecodeString(settingRoot)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("the epoch root key of the settings is not 32 bytes of hex")
	}
	return []ed25519.PublicKey{key}, nil
}

func isDefaultServer(address string) bool {
	got, err := parseServer(strings.TrimSpace(address))
	if err != nil {
		return true
	}
	want, err := parseServer(DefaultServer)
	if err != nil {
		return true
	}
	return got == want
}

type serverKey struct{ scheme, host, port string }

func parseServer(address string) (serverKey, error) {
	u, err := url.Parse(address)
	if err != nil {
		return serverKey{}, err
	}
	key := serverKey{
		scheme: strings.ToLower(u.Scheme),
		host:   strings.TrimSuffix(strings.ToLower(u.Hostname()), "."),
		port:   u.Port(),
	}
	if key.port == "" && key.scheme == "https" {
		key.port = "443"
	}
	return key, nil
}

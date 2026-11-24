package epochtrust

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
)

// DefaultServer is the address of the server of the build, the only one the embedded roots are for.
const DefaultServer = "https://sync.numen.md"

func getEmbeddedRoots() []ed25519.PublicKey { return []ed25519.PublicKey{} }

// GetRoots returns the roots that count for the server at address: the embedded ones, and the root key of the
// plugin setting (hex) when the person typed another server than the default one. A setting never adds a root
// for the default server.
func GetRoots(address, settingRoot string) ([]ed25519.PublicKey, error) {
	roots := getEmbeddedRoots()
	settingRoot = strings.TrimSpace(settingRoot)
	if settingRoot == "" || isDefaultServer(address) {
		return roots, nil
	}
	key, err := hex.DecodeString(settingRoot)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return roots, fmt.Errorf("the epoch root key of the settings is not 32 bytes of hex")
	}
	return append(roots, key), nil
}

func isDefaultServer(address string) bool {
	return strings.EqualFold(strings.TrimRight(address, "/"), DefaultServer)
}

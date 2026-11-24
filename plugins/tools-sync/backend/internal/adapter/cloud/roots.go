package cloud

import (
	"crypto/ed25519"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
)

// Roots are the public keys that may sign the certificate of the epoch key of a server.
type Roots []ed25519.PublicKey

// GetRoots returns the root keys that vouch for the restore epochs of the server at address: the ones of the build, and
// the key of the setting only for a server other than the default one. A setting that is not a key is the error, with
// the keys of the build.
func GetRoots(address, setting string) (Roots, error) {
	return epochtrust.GetRoots(address, setting)
}

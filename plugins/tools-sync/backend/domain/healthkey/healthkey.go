// Package healthkey builds the keys of the health items of the plugin.
package healthkey

import (
	"crypto/sha256"
	"encoding/hex"
)

// MaxLength is the longest key the host takes.
const MaxLength = 32

const tokenLength = 8

// MaxPrefixLength is the longest prefix Create takes: the prefix, a hyphen and the token fill MaxLength.
const MaxPrefixLength = MaxLength - 1 - tokenLength

// Create returns the key of the item that prefix names for the vault or request id: the prefix, a hyphen and
// eight hex digits of the SHA-256 of id, so ids of any length give distinct short keys.
func Create(prefix, id string) string {
	sum := sha256.Sum256([]byte(id))
	return prefix + "-" + hex.EncodeToString(sum[:])[:tokenLength]
}

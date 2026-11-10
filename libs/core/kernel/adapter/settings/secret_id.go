package settings

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// CreateSecretID returns the id of a new secret: the letter s and eight hexadecimal digits from the
// system's random source.
func CreateSecretID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a secret: %w", err)
	}
	return "s" + hex.EncodeToString(random[:]), nil
}

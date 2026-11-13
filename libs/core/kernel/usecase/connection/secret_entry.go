package connection

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// secretFieldType is the type of a field whose value is kept in the key store.
const secretFieldType = "secret"

const (
	entryPrefix          = "connection/"
	fingerprintSeparator = ":"
)

// GetEntryName returns the name the value of a secret field is stored under in the key store.
func GetEntryName(id, key string) string { return entryPrefix + id + "/" + key }

func getFingerprint(typeID, address string) string {
	sum := sha256.Sum256([]byte(typeID + "\x00" + address))
	return hex.EncodeToString(sum[:])
}

func encodeSecret(fingerprint, value string) string {
	return fingerprint + fingerprintSeparator + value
}

func hasFingerprint(entry, fingerprint string) bool {
	stored, _, found := strings.Cut(entry, fingerprintSeparator)
	return found && subtle.ConstantTimeCompare([]byte(stored), []byte(fingerprint)) == 1
}

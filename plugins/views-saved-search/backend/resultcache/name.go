package resultcache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	extension    = ".json"
	pathPrefix   = "path-"
	pathHashSize = 16
)

// GetFileName returns the name of the cache file of a saved search note: its id, or for a note without an
// id or with an id that cannot name a file of the store, the first 16 hex digits of the SHA-256 of its
// path.
func GetFileName(noteID, notePath string) string {
	if isFileName(noteID) {
		return noteID + extension
	}
	return pathPrefix + HashText(notePath)[:pathHashSize] + extension
}

// HashText returns the hex SHA-256 of the text.
func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func isFileName(noteID string) bool {
	return noteID != "" && !strings.HasPrefix(noteID, ".") && !strings.ContainsAny(noteID, "/\\:\x00")
}

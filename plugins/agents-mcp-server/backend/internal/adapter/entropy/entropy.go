// Package entropy provides cryptographically secure random bytes.
package entropy

import "crypto/rand"

// Read fills b with cryptographically secure random bytes.
func Read(b []byte) (int, error) {
	return rand.Read(b)
}

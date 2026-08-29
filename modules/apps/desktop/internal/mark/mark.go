// Package mark mints the marks a card is known by.
//
// A mark says which card this is and nothing else. It is minted once, and it
// travels with the card: the same card moved to another deck or another vault
// is the same mark, and nothing about it is recomputed.
package mark

import "crypto/rand"

// Crockford base32 in lower case: no i, l, o or u, so that a mark read aloud or
// retyped from a file does not turn into a different one.
const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// Length is how many characters a mark is. Ten of this alphabet is 1.1 × 10¹⁵
// marks, so a vault of a hundred thousand cards meets a collision about once in
// two hundred thousand vaults.
const Length = 10

// New mints a mark.
func New() (string, error) {
	var b [Length]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	out := make([]byte, Length)
	for i, v := range b {
		// The alphabet is 32 characters and a byte holds eight of them evenly,
		// so every character is as likely as every other.
		out[i] = alphabet[v&0x1f]
	}
	return string(out), nil
}

// Valid reports whether s could have been minted here. It is strict: a mark is
// read as one only at that length and in that alphabet, and anything else at
// the end of a heading is heading text.
func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	for i := range len(s) {
		found := false
		for j := range len(alphabet) {
			if s[i] == alphabet[j] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

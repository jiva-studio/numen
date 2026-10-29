// Package cardid generates the identifiers that identify a card, and checks
// whether a string is one.
//
// The identifier identifies the card and carries no other information. It is
// generated once and moves with the card: the same card moved to another deck
// or vault keeps the same identifier, which is never recomputed. A card's
// heading writes it after a caret, where it is called the card id.
package cardid

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
)

// Crockford base32 in lower case, without i, l, o or u, which prevents an identifier
// read aloud or retyped from a file from becoming a different identifier.
const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// Length is how many characters an identifier is. Ten of this alphabet is
// 1.1 × 10¹⁵ identifiers, so a vault of a hundred thousand cards meets a
// collision about once in two hundred thousand vaults.
const Length = 10

// isAlphabet holds, for each byte, whether it is a character of the alphabet.
var isAlphabet = createTable(alphabet)

// gomutants:disable-next-line INTEGER_INCREMENT reason="a byte indexes the table, so a 257th slot is never read"
func createTable(characters string) (table [256]bool) {
	for i := range len(characters) {
		table[characters[i]] = true
	}
	return table
}

// New mints an identifier.
func New() ID {
	var b [Length]byte
	// rand.Read aborts the program when the system has no randomness; it never
	// returns an error.
	_, _ = clock.ReadEntropy(b[:])
	out := make([]byte, Length)
	for i, v := range b {
		// The alphabet is 32 characters and a byte holds eight of them evenly,
		// so every character is as likely as every other.
		out[i] = alphabet[v&0x1f]
	}
	return ID(out)
}

// IsValid reports whether s could have been minted here. It is strict: an
// identifier is read as one only at that length and in that alphabet, and
// anything else at the end of a heading is heading text.
func IsValid(s ID) bool {
	if len(s) != Length {
		return false
	}
	for i := range len(s) {
		if !isAlphabet[s[i]] {
			return false
		}
	}
	return true
}

// ID identifies a card for as long as it exists. It is generated once and
// moves with the card: the same card moved to another deck or vault keeps the
// same identifier, which is never recomputed. A card's heading writes it after
// a caret, where it is called the card id.
type ID string

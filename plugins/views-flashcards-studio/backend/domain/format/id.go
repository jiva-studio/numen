package format

import (
	"io"
	"strings"
)

// Crockford base32 in lower case, without i, l, o or u, which prevents an identifier
// read aloud or retyped from a file from becoming a different identifier.
const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// IDLength is how many characters an identifier is. Ten of this alphabet is
// 1.1 × 10¹⁵ identifiers, so a vault of a hundred thousand cards meets a
// collision about once in two hundred thousand vaults.
const IDLength = 10

// CreateID mints an identifier from the random bytes that readEntropy fills in.
func CreateID(readEntropy func([]byte)) ID {
	var b [IDLength]byte
	readEntropy(b[:])
	out := make([]byte, IDLength)
	for i, v := range b {
		// The alphabet is 32 characters and a byte holds eight of them evenly,
		// so every character is as likely as every other.
		out[i] = alphabet[v&0x1f]
	}
	return ID(out)
}

// IsValidID reports whether s could have been minted here. It is strict: an
// identifier is read as one only at that length and in that alphabet, and
// anything else at the end of a heading is heading text.
func IsValidID(s ID) bool {
	if len(s) != IDLength {
		return false
	}
	for i := range len(s) {
		if strings.IndexByte(alphabet, s[i]) < 0 {
			return false
		}
	}
	return true
}

// ID identifies a card for as long as it exists and carries no other information. It is
// generated once and moves with the card: the same card moved to another deck or vault keeps
// the same identifier, which is never recomputed. A card's heading writes it after a caret,
// where it is called the card id.
type ID string

// NewEntropyReader returns what fills entropy as a reader, for a ULID generator. No entropy is a
// nil reader, which the generator refuses with ulid.ErrNoEntropy.
func NewEntropyReader(entropy interface{ Fill(b []byte) }) io.Reader {
	if entropy == nil {
		return nil
	}
	return entropyReader{fill: entropy.Fill}
}

type entropyReader struct{ fill func(b []byte) }

func (r entropyReader) Read(b []byte) (int, error) {
	r.fill(b)
	return len(b), nil
}

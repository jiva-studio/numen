// Package ulid generates the identifiers a vault carries.
//
// A ULID's lexicographic order is its chronological order, which tells the
// moment anything is sorted or merged.
package ulid

import (
	"crypto/rand"
	"errors"
	"time"
)

// Crockford base32: no I, L, O or U, so that an identifier read aloud or
// retyped from a file does not turn into a different one.
const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// New returns a ULID for the given instant: 48 bits of milliseconds followed by
// 80 bits of randomness, encoded as 26 characters.
func New(t time.Time) (string, error) {
	var b [16]byte
	ms := uint64(t.UTC().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := rand.Read(b[6:]); err != nil {
		return "", err
	}

	out := make([]byte, 26)
	// 26 base32 characters carry 130 bits; the value is left-padded into them.
	var carry uint16
	bits := 0
	idx := 25
	for i := 15; i >= 0; i-- {
		carry |= uint16(b[i]) << bits
		bits += 8
		for bits >= 5 {
			out[idx] = alphabet[carry&0x1f]
			idx--
			carry >>= 5
			bits -= 5
		}
	}
	for idx >= 0 {
		out[idx] = alphabet[carry&0x1f]
		idx--
		carry >>= 5
	}
	return string(out), nil
}

// Valid reports whether s could have been produced by New. It is deliberately
// strict: an identifier that is almost right points at a corrupted file rather
// than at something to be tolerated.
func Valid(s string) bool {
	if len(s) != 26 {
		return false
	}
	for i := 0; i < len(s); i++ {
		found := false
		for j := 0; j < len(alphabet); j++ {
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

// ErrInvalid is returned by callers that read an identifier from a file.
var ErrInvalid = errors.New("not a valid ULID")

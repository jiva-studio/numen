// Package ulid generates the identifiers a vault uses.
//
// A ULID's lexicographic order is its chronological order, which orders identifiers
// by creation time when sorted or merged.
package ulid

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Crockford base32: no I, L, O or U, which keeps an identifier unambiguous when read
// aloud or retyped from a file.
const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ErrInvalid is returned by callers that read an identifier from a file.
var ErrInvalid = errors.New("not a valid ULID")

// ErrNoEntropy is returned by a generator that was given no entropy source.
var ErrNoEntropy = errors.New("no entropy source for a ULID")

// Generator creates identifiers whose random bits are read from one entropy source.
type Generator struct {
	entropy io.Reader
}

// NewGenerator returns a generator that reads the random bits of each identifier from
// entropy. A generator over crypto/rand.Reader is safe for concurrent use.
func NewGenerator(entropy io.Reader) Generator {
	return Generator{entropy: entropy}
}

// NewID returns a ULID for the given instant: 48 bits of milliseconds followed by
// 80 bits read from the entropy source, encoded as 26 characters.
func (g Generator) NewID(t time.Time) (string, error) {
	if g.entropy == nil {
		return "", ErrNoEntropy
	}
	var b [16]byte
	ms := uint64(t.UTC().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := io.ReadFull(g.entropy, b[6:]); err != nil {
		return "", fmt.Errorf("read the random bits of a ULID: %w", err)
	}
	return encode(b), nil
}

func encode(b [16]byte) string {
	out := make([]byte, 26)
	// 26 base32 characters carry 130 bits; the value is left-padded into them.
	var carry uint16
	bits := 0
	idx := 25
	for i := 15; i >= 0; i-- {
		carry |= uint16(b[i]) << bits
		bits += 8
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_INCREMENT reason="uint16 carry holds 13 bits when a 5-bit flush is deferred"
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
		// gomutants:disable-next-line INVERT_BITWISE_ASSIGNMENTS,REMOVE_SELF_ASSIGNMENTS,INTEGER_DECREMENT,INTEGER_INCREMENT reason="the loop runs once for idx 0 and carry is unread afterwards"
		carry >>= 5
	}
	return string(out)
}

// Valid reports whether s could have been produced by NewID. It is deliberately
// strict: an identifier that is almost right points at a corrupted file.
func Valid(s string) bool {
	if len(s) != 26 {
		return false
	}
	for i := range len(s) {
		if strings.IndexByte(alphabet, s[i]) < 0 {
			return false
		}
	}
	return true
}

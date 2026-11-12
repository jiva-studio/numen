// Package ulid exposes the identifier generator of the notes library to plugins.
package ulid

import (
	"io"

	notesulid "github.com/jiva-studio/numen/libs/note/ulid"
)

// Generator creates identifiers whose random bits are read from one entropy source.
type Generator = notesulid.Generator

// ErrNoEntropy is returned by a generator that was given no entropy source.
var ErrNoEntropy = notesulid.ErrNoEntropy

// NewGenerator returns a generator that reads the random bits of each identifier from
// entropy, usually crypto/rand.Reader.
func NewGenerator(entropy io.Reader) Generator {
	return notesulid.NewGenerator(entropy)
}

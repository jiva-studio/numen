// Package ulid exposes the identifier generator of the notes library to plugins.
package ulid

import (
	"crypto/rand"
	"io"
	"time"

	notesulid "github.com/jiva-studio/numen/libs/notes/ulid"
)

func init() {
	notesulid.DefaultEntropy = rand.Reader
}

func New(t time.Time, entropy ...io.Reader) string {
	if len(entropy) == 0 {
		return notesulid.New(t, rand.Reader)
	}
	return notesulid.New(t, entropy...)
}

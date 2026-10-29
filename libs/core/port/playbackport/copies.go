// Package playbackport declares what the core needs to play a file of the vault from a copy a plugin
// keeps of it.
package playbackport

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrNoProxy is a file no enabled plugin keeps a copy of.
var ErrNoProxy = errors.New("no copy of the file is kept")

// Copy is an open copy of a file from which a part can be read. The opener
// closes it.
type Copy struct {
	Body      io.ReadSeekCloser
	Size      int64
	MediaType string
}

// Copies are the copies the enabled plugins keep of the vault's files, each played in place of its
// file. A file is matched by its content type, and a note also by its note type; a file that is no
// note has the empty note type.
type Copies interface {
	// IsCopyingNotes reports whether an enabled plugin keeps copies of notes of some type.
	IsCopyingNotes() bool
	// IsCopied reports whether an enabled plugin keeps copies of files of the type of the path, or
	// of notes of the note type.
	IsCopied(path string, noteType domain.NoteType) bool
	// OpenProxy opens the copy kept of the file at the path whose content has the hash, or returns
	// ErrNoProxy.
	OpenProxy(ctx context.Context, v domain.Vault, path string, noteType domain.NoteType, hash string) (Copy, error)
}

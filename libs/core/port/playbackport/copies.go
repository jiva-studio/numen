// Package playbackport declares what the core needs to play a file of the vault from a copy a plugin
// keeps of it.
package playbackport

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrNoCopy is a file no enabled plugin keeps a copy of.
var ErrNoCopy = errors.New("no copy of the file is kept")

// Copy is a copy open to read a part of. Whoever opens it closes it.
type Copy struct {
	Body      io.ReadSeekCloser
	Size      int64
	MediaType string
}

// Copies are the copies the enabled plugins keep of the vault's files, each played in place of its
// file.
type Copies interface {
	// IsCopied reports whether an enabled plugin keeps copies of files of the type of the path.
	IsCopied(path string) bool
	// OpenCopy opens the copy kept of the file at the path whose content has the hash, or returns
	// ErrNoCopy.
	OpenCopy(ctx context.Context, v domain.Vault, path, hash string) (Copy, error)
}

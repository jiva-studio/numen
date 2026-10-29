package vaultport

import (
	"context"
	"errors"
	"io"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// ErrNotANote is what a path gets when there is a file at it and the vault does
// not hold that file as a note: an attachment, an export, a folder some tool
// keeps its state in, whatever the vault's own rules leave alone.
//
// It is its own answer and not ErrNotExist. A file that is there and a path
// with nothing at it are different facts, and a caller acts on them
// differently.
var ErrNotANote = edit.ErrNotANote

// ErrNotExist is what a path with nothing at it gets.
var ErrNotExist = edit.ErrNotExist

// ErrExist is what a path that already holds a file gets when a write requires an unoccupied path.
var ErrExist = edit.ErrExist

// ErrOutside is what a path that does not stay in the vault gets. It is not
// "no note": a caller asking for something outside has made a mistake worth
// hearing about, and "no such note" would hide it.
var ErrOutside = errors.New("not a path inside the vault")

// IsNoNote reports whether an error means there is no note at the path: nothing
// exists there, or the file there is one the vault ignores.
//
// A caller that needs to act on a note checks this. The two cases are
// distinguished by the two sentinel errors, for callers that handle them
// differently.
func IsNoNote(err error) bool {
	return edit.IsNoNote(err)
}

// VaultReader is one vault as seen from outside: something that can be walked
// and read. The filesystem is one implementation; a fixture in memory is
// another.
type VaultReader interface {
	// WalkFiles reports every source the vault holds, in unspecified order, each
	// saying which kind it is. The service folder is not reported, and neither
	// is a file of no kind the application reads.
	WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error
	// List reports the entries of one folder, without descending: the files of
	// every kind and the folders under it. The empty path is the vault root.
	//
	// What a walk leaves out is left out here as well. Folders come first,
	// then files, each by name compared without regard to case, and that is
	// the order to draw them in.
	//
	// ErrOutside for a path that leaves the vault, fs.ErrNotExist for a folder
	// that is not there and for a path that holds a file.
	List(ctx context.Context, folder string) ([]domain.Entry, error)
	// Read returns the bytes of one file, addressed by a path a walk reported.
	Read(ctx context.Context, path string) ([]byte, error)
	// Open returns a file for partial reads. A recording is hours long and is
	// played from the middle, so reading the whole file into memory for every
	// listener is not acceptable.
	//
	// The errors are the same as Read. The caller closes the file.
	Open(ctx context.Context, path string) (io.ReadSeekCloser, error)
	// StatFile returns what a walk reports about one path: its kind, size and
	// modification time. It reads no bytes, so it is the call for a file the
	// caller does not want to open.
	//
	// It returns fs.ErrNotExist when there is nothing at the path, and
	// ErrNotANote when the file there is one the vault ignores.
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
}

// VaultReaders opens a reader for a given vault. Which vault a use case works
// on is decided while it runs.
type VaultReaders interface {
	Open(v domain.Vault) (VaultReader, error)
}

// IsNote reports whether the file at the path is a note, by its extension.
func IsNote(path string) bool {
	return strings.EqualFold(pathpkg.Ext(path), domain.NoteExtension)
}

// ReadNoteType returns the type declared in the frontmatter of the note at the
// path, read from the current file, so a note just written has its type before
// the index holds it. A file that is not a note has no type and is not read.
func ReadNoteType(ctx context.Context, readers VaultReaders, v domain.Vault, path string) (domain.NoteType, error) {
	if !IsNote(path) {
		return "", nil
	}
	reader, err := readers.Open(v)
	if err != nil {
		return "", err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	return markdown.Parse(domain.FileStat{}, raw).Type, nil
}

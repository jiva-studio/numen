package edit

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// MaxBytes is the most a note may be and still be read here.
//
// A megabyte of prose is about a quarter of a million words. A larger file is
// a pasted export or an error. The size is checked before the file is opened.
const MaxBytes = 1 << 20

var (
	// ErrStale is returned by a write when the file to be replaced differs from
	// the content the caller read.
	ErrStale = errors.New("the note changed since it was read")
	// ErrNotANote is returned for a path that holds a file the vault does not
	// treat as a note.
	ErrNotANote = errors.New("not a note this vault holds")
	// ErrNoNote is returned when the vault holds no note at the given path.
	ErrNoNote = errors.New("the vault holds no note at this path")
	// ErrUnlevelled is returned when a file was written but the index could not
	// be updated. The vault holds the written content, and the next scan
	// updates the index for the path.
	ErrUnlevelled = errors.New("the vault was written and the index was not brought level with it")
	// ErrTooLarge is returned for a note over MaxBytes. Nothing is written. The
	// limit applies to the whole file to be stored, as it does for a read.
	ErrTooLarge = errors.New("this is more text than a note is written with")
	// ErrNotExist is returned for a path with no file.
	ErrNotExist = fs.ErrNotExist
	// ErrExist is returned when a write requires an unoccupied path and the path already holds a file.
	ErrExist = fs.ErrExist
)

// ErrBodyUnwritable is a body that opens with the frontmatter delimiter.
var ErrBodyUnwritable = markdown.ErrBodyUnwritable

// ErrUnreadable is returned for a note whose frontmatter is not YAML. A write
// that finds it leaves the file unchanged, because repairing the block would
// require guessing the author's intent.
var ErrUnreadable = markdown.ErrUnreadable

// ErrInline, ErrUnterminated and ErrAnchored identify frontmatter in which a
// single key cannot be changed: a block written on one line, a block that is
// never closed, and a block with a YAML anchor. Such a note is left unchanged.
var (
	ErrInline       = markdown.ErrInline
	ErrUnterminated = markdown.ErrUnterminated
	ErrAnchored     = markdown.ErrAnchored
)

// IsNoNote reports whether the error means the vault holds no note at the
// path: no file exists there, or the file is not a note.
func IsNoNote(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrNotANote)
}

// WrapUnlevelled wraps an index update failure in ErrUnlevelled, which
// distinguishes it from a write that failed. The paths are the ones the index
// update covered.
func WrapUnlevelled(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrUnlevelled, strings.Join(paths, ", "), err)
}

// MapMissingNote returns ErrNoNote when the vault holds no note at the path,
// and any other err unchanged.
func MapMissingNote(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNoNote
	}
	return err
}

// CheckSize holds a file to the most it may be: its frontmatter and its prose together, as they are
// about to go to disk. Zero holds it to nothing.
func CheckSize(path string, size, maxSize int) error {
	if maxSize <= 0 || size <= maxSize {
		return nil
	}
	return fmt.Errorf("write %s: %w: %d bytes, and %d is the most", path, ErrTooLarge, size, maxSize)
}

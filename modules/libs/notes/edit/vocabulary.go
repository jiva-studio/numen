package edit

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

// MaxBytes is the most a note may be and still be read here.
//
// A note is prose somebody wrote, and a megabyte of it is a quarter of a million words. Something
// larger is a pasted export or a mistake. The size is asked of the file before it is opened, so a
const MaxBytes = 1 << 20

var (
	// ErrStale is what a write says when the file it was about to replace is no longer what the
	// caller read.
	ErrStale = errors.New("the note changed since it was read")
	// ErrNotANote is what a path gets when there is a file at it and the vault does not hold it as
	// a note.
	ErrNotANote = errors.New("not a note this vault holds")
	// ErrNoNote is what an operation gets when the vault holds no note at the path it was given.
	ErrNoNote = errors.New("the vault holds no note at this path")
	// ErrUnlevelled is a file that was written and an index that could not be brought up to date
	// with it. The vault holds what the write put there, and the path is levelled again at the next
	// scan.
	ErrUnlevelled = errors.New("the vault was written and the index was not brought level with it")
	// ErrTooLarge is what a note over MaxBytes gets. Nothing is written. The bound is measured
	// against the whole file about to be stored, which is what a read measures.
	ErrTooLarge = errors.New("this is more text than a note is written with")
)

// ErrBodyUnwritable is a body that opens with the frontmatter delimiter.
var ErrBodyUnwritable = markdown.ErrBodyUnwritable

// ErrUnreadable is a note whose frontmatter is not YAML. A write discovers it on the way past and
// leaves the file alone: repairing the block means guessing at what the person wrote.
var ErrUnreadable = markdown.ErrUnreadable

// ErrInline, ErrUnterminated and ErrAnchored are frontmatter one key cannot be changed in: a block
// written on one line, a block that is never closed, and a block carrying a YAML anchor. Such a
// note is left alone.
var (
	ErrInline       = markdown.ErrInline
	ErrUnterminated = markdown.ErrUnterminated
	ErrAnchored     = markdown.ErrAnchored
)

// IsNoNote reports whether the error says the vault holds no note at the path: nothing is there, or
// what is there is not a note.
func IsNoNote(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrNotANote)
}

// WrapUnlevelled is what bringing the index up to date came to, said so that a caller can tell it
// from a write that never landed. The paths are the ones the index was asked about.
func WrapUnlevelled(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrUnlevelled, strings.Join(paths, ", "), err)
}

// MapMissingNote is ErrNoNote where the vault holds no note at the path, and the error as it arrived
// otherwise.
func MapMissingNote(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNoNote
	}
	return err
}

// CheckSize holds a file to the most it may be: its frontmatter and its prose together, as they are
// about to go to disk. Zero holds it to nothing.
func CheckSize(path string, size, bound int) error {
	if bound <= 0 || size <= bound {
		return nil
	}
	return fmt.Errorf("write %s: %w: %d bytes, and %d is the most", path, ErrTooLarge, size, bound)
}

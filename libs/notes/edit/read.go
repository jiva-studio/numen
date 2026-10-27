package edit

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// ReadOutcome is how a read ended. Each one leaves the caller with something different to do: a
// missing note is created by writing it, a note that is too large is opened in something else, and
// a file that is not a note or not text is left alone.
type ReadOutcome string

const (
	// Ok is the body coming back.
	Ok ReadOutcome = "ok"
	// Missing is a path with no file behind it.
	Missing ReadOutcome = "missing"
	// NotANote is a file the vault does not hold as one: an attachment, an export, whatever the
	// vault's own rules leave alone.
	NotANote ReadOutcome = "not a note"
	// NotText is a file that is not valid UTF-8.
	NotText ReadOutcome = "not text"
	// TooLarge is a file over the bound.
	TooLarge ReadOutcome = "too large"
	// Unreadable is a note whose frontmatter cannot be read. Such a note can be neither read nor
	// written from here.
	Unreadable ReadOutcome = "unreadable"
)

// Reader is what a read asks of the files.
type Reader interface {
	// StatFile says what the file is without opening it. It returns fs.ErrNotExist for a file
	// that is not there and ErrNotANote for one the vault does not hold as a note.
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	// Read returns the bytes of the file.
	Read(ctx context.Context, path string) ([]byte, error)
}

// Contents is one note as a read hands it over.
type Contents struct {
	Path    string
	Outcome ReadOutcome
	// Body is the prose below the frontmatter, with every line break written as one \n. It is empty
	// for every outcome but Ok.
	Body string
	// FileStat is what the file was when it was asked about, which is before its bytes were read.
	// size still says what it was.
	FileStat domain.FileStat
}

// Read hands over the prose of the note at path, refusing what is larger than MaxBytes.
//
// An error is the vault being out of reach. What is wrong with the note itself is an outcome, and
// the caller is told which one.
func Read(ctx context.Context, files Reader, path string) (Contents, error) {
	out := Contents{Path: path}

	// The vault says what is at a path without opening it: a note, a file it leaves alone, or
	// nothing.
	ref, err := files.StatFile(ctx, path)
	target := err == nil
	switch {
	case target:
		out.FileStat = ref
		// A vault holds several kinds of source and this reads one of them.
		if ref.Kind != domain.KindNote {
			out.Outcome = NotANote
			return out, nil
		}
		if ref.Size > MaxBytes {
			out.Outcome = TooLarge
			return out, nil
		}
	case errors.Is(err, ErrNotANote):
		out.Outcome = NotANote
		return out, nil
	case !errors.Is(err, fs.ErrNotExist):
		return Contents{}, fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := files.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = Missing
		return out, nil
	case !target && err == nil:
		out.Outcome = NotANote
		return out, nil
	case err != nil:
		return Contents{}, fmt.Errorf("read %s: %w", path, err)
	}

	// The body goes out in a string field. A browser turns an ill-formed sequence into U+FFFD, and
	// the next save puts those characters where the person's bytes were.
	if !utf8.Valid(raw) {
		out.Outcome = NotText
		return out, nil
	}

	doc, err := markdown.Open(raw)
	if errors.Is(err, markdown.ErrUnreadable) {
		out.Outcome = Unreadable
		return out, nil
	}
	if err != nil {
		return Contents{}, fmt.Errorf("read %s: %w", path, err)
	}
	out.Body = markdown.Normalise(doc.Body())
	out.Outcome = Ok
	return out, nil
}

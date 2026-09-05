package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// MaxBytes is the most a note may be and still be read here.
//
// A note is prose somebody wrote, and a megabyte of it is a quarter of a
// million words. Something larger is a pasted export or a mistake. The size is
// asked of the file before it is opened, so a note over the bound is refused
// without its bytes being read.
const MaxBytes = 1 << 20

// ReadOutcome is how a read ended. Each one leaves the caller with something
// different to do: a missing note is created by writing it, a note that is too
// large is opened in something else, and a file that is not a note or not text
// is left alone.
type ReadOutcome string

const (
	// Ok is the body coming back.
	Ok ReadOutcome = "ok"
	// Missing is a path with no file behind it.
	Missing ReadOutcome = "missing"
	// NotANote is a file the vault does not hold as one: an attachment, an
	// export, whatever the vault's own rules leave alone.
	NotANote ReadOutcome = "not a note"
	// NotText is a file that is not valid UTF-8.
	NotText ReadOutcome = "not text"
	// TooLarge is a file over MaxBytes.
	TooLarge ReadOutcome = "too large"
	// Unreadable is a note whose frontmatter cannot be read. Such a note can be
	// neither read nor written from here.
	Unreadable ReadOutcome = "unreadable"
)

// Contents is one note as a read hands it over.
type Contents struct {
	Path    string
	Outcome ReadOutcome
	// Body is the prose below the frontmatter, with every line break written as
	// one \n. It is empty for every outcome but Ok.
	Body string
	// Fingerprint is what the file was when it was asked about, which is before
	// its bytes were read. It is set for any path the vault holds, whatever kind
	// it holds it as, so a note refused on its size still says what it was.
	Fingerprint domain.Fingerprint
}

// Read hands over the prose of one note.
//
// The window and an agent read through here, so what is refused to one is
// refused to the other, and the rules for it are written once.
type Read struct {
	Readers port.VaultReaders
}

// NewRead is what a note is read out of: the vault it stands in.
//
// It is named here because the window and an agent each build this where they
// stand, and neither has anything to hand a person without it.
func NewRead(readers port.VaultReaders) Read {
	return Read{Readers: readers}
}

// Execute reads the note at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
// is an outcome, and the caller is told which one.
func (u Read) Execute(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	reader, err := u.Readers.Open(v)
	if err != nil {
		return Contents{}, err
	}
	out := Contents{Path: path}

	// The vault says what is at a path without opening it: a note, a file it
	// leaves alone, or nothing.
	ref, err := reader.Stat(ctx, path)
	held := err == nil
	switch {
	case held:
		out.Fingerprint = ref
		// A vault holds several kinds of source and this reads one of them.
		if ref.Kind != domain.KindNote {
			out.Outcome = NotANote
			return out, nil
		}
		if ref.Size > MaxBytes {
			out.Outcome = TooLarge
			return out, nil
		}
	case errors.Is(err, port.ErrNotANote):
		out.Outcome = NotANote
		return out, nil
	case !errors.Is(err, fs.ErrNotExist):
		return Contents{}, fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = Missing
		return out, nil
	case !held && err == nil:
		out.Outcome = NotANote
		return out, nil
	case err != nil:
		return Contents{}, fmt.Errorf("read %s: %w", path, err)
	}

	// The body goes out in a string field. A browser turns an ill-formed
	// sequence into U+FFFD, and the next save puts those characters where the
	// person's bytes were.
	if !utf8.Valid(raw) {
		out.Outcome = NotText
		return out, nil
	}

	doc, err := markdown.Open(raw)
	if err != nil {
		out.Outcome = Unreadable
		return out, nil
	}
	out.Body = markdown.Normalised(doc.Body())
	out.Outcome = Ok
	return out, nil
}

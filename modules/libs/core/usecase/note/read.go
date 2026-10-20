package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
)

// MaxBytes is the most a note may be and still be read here.
const MaxBytes = edit.MaxBytes

// ReadOutcome is how a read ended.
type ReadOutcome = edit.ReadOutcome

const (
	// Ok is the body coming back.
	Ok = edit.Ok
	// Missing is a path with no file behind it.
	Missing = edit.Missing
	// NotANote is a file the vault does not hold as one.
	NotANote = edit.NotANote
	// NotText is a file that is not valid UTF-8.
	NotText = edit.NotText
	// TooLarge is a file over MaxBytes.
	TooLarge = edit.TooLarge
	// Unreadable is a note whose frontmatter cannot be read.
	Unreadable = edit.Unreadable
)

// Contents is one note as a read hands it over.
type Contents = edit.Contents

// Read hands over the prose of one note.
//
// The window and an agent read through here, so what is refused to one is
// refused to the other, and the rules for it are written once.
type Read struct {
	readers port.VaultReaders
}

var errNotOpen = errors.New("this vault's files are not open")

// NewRead is what a note is read out of: the vault it stands in.
func NewRead(readers port.VaultReaders) Read {
	return Read{readers: readers}
}

// Execute reads the note at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
// is an outcome, and the caller is told which one.
func (u Read) Execute(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	if u.readers == nil {
		return Contents{}, errNotOpen
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return Contents{}, err
	}
	return edit.Read(ctx, reader, path)
}

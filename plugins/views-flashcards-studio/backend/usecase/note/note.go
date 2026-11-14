// Package note is what the use cases of the plugin do to a note held in a vault: reading its prose
// and listing what it is joined to.
//
// The rules are the notes module's; this package puts them over the ports of the plugin.
package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// The notes module's words for a read and an edit.
type (
	// ReadOutcome is how a read ended.
	ReadOutcome = edit.ReadOutcome
	// Contents is one note as returned by a read.
	Contents = edit.Contents
	// IndexUpdater brings the named notes up to date in the index.
	IndexUpdater = edit.IndexUpdater
)

const (
	// MaxBytes is the most a note may be and still be read here.
	MaxBytes = edit.MaxBytes
)

// The outcomes of reading a note.
const (
	Ok         = edit.Ok
	Missing    = edit.Missing
	NotANote   = edit.NotANote
	NotText    = edit.NotText
	TooLarge   = edit.TooLarge
	Unreadable = edit.Unreadable
)

// The errors of reading and writing a note.
var (
	ErrNoNote         = edit.ErrNoNote
	ErrIndexOutdated  = edit.ErrIndexOutdated
	ErrTooLarge       = edit.ErrTooLarge
	ErrBodyUnwritable = edit.ErrBodyUnwritable
)

// Read returns the prose of one note.
type Read struct {
	readers ports.VaultReaders
}

func NewRead(readers ports.VaultReaders) Read { return Read{readers: readers} }

// Execute reads the note at path. An error is the vault being out of reach; what is wrong with the
// note itself is an outcome.
func (u Read) Execute(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return Contents{}, err
	}
	return edit.Read(ctx, reader, path)
}

// ShowLinks gathers what one note is joined to.
type ShowLinks struct {
	links ports.LinkQueries
}

// NewShowLinks returns a ShowLinks that serves both halves from links.
func NewShowLinks(links ports.LinkQueries) ShowLinks { return ShowLinks{links: links} }

// Execute returns both halves of what the note is joined to.
func (u ShowLinks) Execute(ctx context.Context, v domain.Vault, path string) (ports.NoteLinks, error) {
	links, err := u.links.GetLinks(ctx, v.ID, path)
	if err != nil {
		return ports.NoteLinks{}, err
	}
	backlinks, err := u.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return ports.NoteLinks{}, err
	}
	return ports.NoteLinks{Links: links, Backlinks: backlinks}, nil
}

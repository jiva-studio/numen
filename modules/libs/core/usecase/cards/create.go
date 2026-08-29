package cards

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// The two frontmatter keys a deck and a stencil are made with: what the note
// is, and what a card cut by it is asked for.
const (
	typeKey   = "type"
	fieldsKey = "fields"
)

// Create makes a deck or a stencil.
//
// One key says what a note is, and the file carries it from the moment it
// exists: a deck made here is a deck to everything that reads the vault, before
// anybody has written a card into it.
type Create struct {
	Writers port.VaultWriters
	// Index brings the new file up to date, so that a caller which makes a deck
	// and lists the vault's decks in the next breath finds it.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	// Extension is what a new file is filed under. Empty means markdown.
	Extension string
	// Now is when this is happening. An identifier carries it.
	Now func() time.Time
}

// New is what to make.
type New struct {
	Title string
	Body  string
	// Folder is where in the vault it goes, relative to the root. Empty is the
	// root itself: the application does not arrange anyone's folders.
	Folder string
	// Fields is what a stencil declares, in the order a person is asked for
	// them. A deck declares none.
	Fields []string
}

// Made is the file that now exists.
type Made struct {
	Path       string
	Identifier string
	Title      string
}

// Deck makes a deck of no cards.
func (u Create) Deck(ctx context.Context, v domain.Vault, in New) (Made, error) {
	in.Fields = nil
	return u.make(ctx, v, domain.TypeDeck, in)
}

// ErrNoFields is what making a stencil says when it was given no field. The
// first field is what a card cut by the stencil is named by.
var ErrNoFields = errors.New("a stencil declares at least one field")

// Stencil makes a stencil declaring the fields it was given.
func (u Create) Stencil(ctx context.Context, v domain.Vault, in New) (Made, error) {
	if len(in.Fields) == 0 {
		return Made{}, fmt.Errorf("%w: %q was given none", ErrNoFields, in.Title)
	}
	return u.make(ctx, v, domain.TypeStencil, in)
}

func (u Create) make(ctx context.Context, v domain.Vault, kind domain.NoteType, in New) (Made, error) {
	title := strings.TrimSpace(in.Title)
	name, exact := domain.Filename(title)
	switch {
	case name == "":
		return Made{}, fmt.Errorf(
			"%w: %q leaves nothing a file can be named after", note.ErrUnnameable, title)
	case strings.ContainsAny(title, "\n\r"):
		return Made{}, fmt.Errorf("%w: %q is more than one line", note.ErrUnnameable, title)
	}
	path := pathpkg.Join(in.Folder, name+u.extension())

	identifier, err := ulid.New(u.now())
	if err != nil {
		return Made{}, err
	}
	doc, err := markdown.Open(markdown.Create(identifier, in.Body))
	if err != nil {
		return Made{}, err
	}
	if err := doc.SetScalar(typeKey, string(kind)); err != nil {
		return Made{}, err
	}
	if len(in.Fields) > 0 {
		if err := doc.SetList(fieldsKey, in.Fields); err != nil {
			return Made{}, err
		}
	}
	// A note is shown by its `title`, else by its filename, so the key is
	// written only where the filename cannot carry the whole name.
	if !exact {
		if err := doc.SetTitle(title); err != nil {
			return Made{}, err
		}
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return Made{}, err
	}
	// Whether the path was free is the filesystem's to answer, at the moment
	// the file is made.
	if err := writer.Create(ctx, path, doc.Bytes()); err != nil {
		return Made{}, err
	}

	// The file is on disk from here on, so what comes back says where it is
	// whether or not the index caught up.
	made := Made{Path: path, Identifier: identifier, Title: title}
	if u.Index == nil {
		return made, nil
	}
	return made, u.Index(ctx, v, []string{path})
}

func (u Create) extension() string {
	if u.Extension == "" {
		return ".md"
	}
	return u.Extension
}

func (u Create) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

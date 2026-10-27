package cards

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// The two frontmatter keys a deck and a card model are made with: what the note
// is, and what a card cut by it is asked for.
const (
	typeKey   = "type"
	fieldsKey = "fields"
)

// Create makes a deck, a card model or a preset.
//
// One key says what a note is, and the file carries it from the moment it
// exists: a deck made here is a deck to everything that reads the vault, before
// anybody has written a card into it.
type Create struct {
	writers ports.VaultWriters
	// index brings the new file up to date, so that a caller which makes a deck
	// and lists the vault's decks in the next breath finds it.
	index note.Levels
	// now is when this is happening. An identifier carries it.
	now ports.Clock
}

// NewCreate is what a deck, a card model or a preset is made through: the vault it
// is written into, what brings the new file level in the index, and what time
// it is.
func NewCreate(writers ports.VaultWriters, index note.Levels, now ports.Clock) Create {
	return Create{writers: writers, index: index, now: now}
}

// WithIndex replaces the index function.
func (u Create) WithIndex(index note.Levels) Create {
	u.index = index
	return u
}

// New is what to make.
type New struct {
	Title string
	Body  string
	// Path is the folder it goes in, relative to the root. Empty is the root
	// itself: the application does not arrange anyone's folders.
	Path string
	// Fields is what a card model declares, in the order a person is asked for
	// them. A deck declares none.
	Fields []string
	// Templates are the ways a card cut by a card model is shown, laid down as its body.
	Templates []format.CardTemplate
}

// CreateNoteResult is the file that now exists.
type CreateNoteResult struct {
	Path  string
	ID    string
	Title string
}

// Deck makes a deck of no cards.
func (u Create) Deck(ctx context.Context, v domain.Vault, in New) (CreateNoteResult, error) {
	in.Fields = nil
	return u.make(ctx, v, format.TypeDeck, in)
}

// ErrNoFields is what making a card model says when it was given no field. The
// first field is what a card cut by the card model is named by.
var ErrNoFields = errors.New("a card model declares at least one field")

// CardModel makes a card model declaring the fields it was given, with its templates laid
// down as its body.
func (u Create) CardModel(ctx context.Context, v domain.Vault, in New) (CreateNoteResult, error) {
	if len(in.Fields) == 0 {
		return CreateNoteResult{}, fmt.Errorf("%w: %q was given none", ErrNoFields, in.Title)
	}
	if len(in.Templates) > 0 {
		body, err := format.CardModelBody("", in.Templates, "")
		if err != nil {
			return CreateNoteResult{}, err
		}
		if len(body) > note.MaxBytes {
			return CreateNoteResult{}, fmt.Errorf(
				"a card model of %d bytes is more than this writes at once, which is %d", len(body), note.MaxBytes)
		}
		in.Body = body
	}
	return u.make(ctx, v, format.TypeCardModel, in)
}

// Preset makes a preset naming none of its settings. A key the file does not
// them until the person moves one.
func (u Create) Preset(ctx context.Context, v domain.Vault, in New) (CreateNoteResult, error) {
	in.Fields = nil
	return u.make(ctx, v, format.TypePreset, in)
}

func (u Create) make(ctx context.Context, v domain.Vault, kind domain.NoteType, in New) (CreateNoteResult, error) {
	title := strings.TrimSpace(in.Title)
	name, exact, err := domain.Filename(title)
	if err != nil {
		return CreateNoteResult{}, err
	}
	path := pathpkg.Join(in.Path, name+domain.NoteExtension)

	identifier := ulid.New(u.now())
	doc, err := markdown.Open(markdown.Create(identifier, in.Body))
	if err != nil {
		return CreateNoteResult{}, err
	}
	if err := doc.SetScalar(typeKey, string(kind)); err != nil {
		return CreateNoteResult{}, err
	}
	if err := doc.SetList(fieldsKey, in.Fields); err != nil {
		return CreateNoteResult{}, err
	}
	// A note is shown by its `title`, else by its filename, so the key is
	// written only where the filename cannot carry the whole name.
	if !exact {
		if err := doc.SetTitle(title); err != nil {
			return CreateNoteResult{}, err
		}
	}

	writer, err := u.writers.Open(v)
	if err != nil {
		return CreateNoteResult{}, err
	}
	// Whether the path was free is the filesystem's to answer, at the moment
	// the file is made.
	if err := writer.Create(ctx, path, doc.Bytes()); err != nil {
		return CreateNoteResult{}, err
	}

	// The file is on disk from here on, so what comes back says where it is
	// whether or not the index caught up.
	made := CreateNoteResult{Path: path, ID: identifier, Title: title}
	if u.index == nil {
		return made, nil
	}
	return made, note.WrapUnlevelled(u.index(ctx, v, []string{path}), path)
}

package cards

import (
	"context"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// The two frontmatter keys written when a deck or a card model is created: the
// note type, and the fields a card of the card model contains.
const (
	typeKey   = "type"
	fieldsKey = "fields"
)

// Create creates a deck, a card model or a preset.
//
// One key states the note type, and the file contains it from the moment it
// exists: a deck created here is a deck for every reader of the vault before
// any card is written into it.
type Create struct {
	writers ports.VaultWriters
	index   edit.IndexUpdater
	now     ports.Clock
	entropy ports.Entropy
}

// NewCreate is what a deck, a card model or a preset is made through: the vault it is written
// into, what updates the index for the new file, what time it is, and the entropy of its
// identifier.
func NewCreate(writers ports.VaultWriters, index edit.IndexUpdater, now ports.Clock, entropy ports.Entropy) Create {
	return Create{writers: writers, index: index, now: now, entropy: entropy}
}

// New is what to make.
type New struct {
	Title     string
	Body      string
	Path      string
	Fields    []string
	Templates []format.CardTemplate
}

// CreateNoteResult is the file that now exists.
type CreateNoteResult struct {
	Path  string
	ID    string
	Title string
}

func (u Create) Deck(ctx context.Context, v domain.Vault, in New) (CreateNoteResult, error) {
	in.Fields = nil
	return u.make(ctx, v, format.TypeDeck, in)
}

// ErrNoFields is returned when a card model is created with no field. The
// first field is the field a card of the card model is named by.
var ErrNoFields = errors.New("a card model declares at least one field")

// CardModel makes a card model declaring the fields it was given, with its templates laid
// down as its body.
func (u Create) CardModel(ctx context.Context, v domain.Vault, in New) (CreateNoteResult, error) {
	if len(in.Fields) == 0 {
		return CreateNoteResult{}, fmt.Errorf("%w: %q was given none", ErrNoFields, in.Title)
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="Slice length is never negative; empty templates generate an empty body identical to the zero value."
	if len(in.Templates) > 0 {
		body, err := format.CardModelBody("", in.Templates, "")
		// gomutants:disable-next-line BRANCH_IF reason="CardModelBody on an empty in-memory preamble does not fail."
		if err != nil {
			return CreateNoteResult{}, err
		}
		if len(body) > edit.MaxBytes {
			return CreateNoteResult{}, fmt.Errorf(
				"a card model of %d bytes is more than this writes at once, which is %d", len(body), edit.MaxBytes)
		}
		in.Body = body
	}
	return u.make(ctx, v, format.TypeCardModel, in)
}

// Preset creates a preset with none of its settings written. A setting absent
// from the file takes its default until the user changes it.
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

	identifier, err := ulid.NewGenerator(format.NewEntropyReader(u.entropy)).NewID(u.now.Now())
	// gomutants:disable-next-line BRANCH_IF reason="The entropy of the machine does not fail."
	if err != nil {
		return CreateNoteResult{}, err
	}
	doc, err := markdown.Open(markdown.Create(identifier, in.Body))
	// gomutants:disable-next-line BRANCH_IF reason="markdown.Create produces valid frontmatter that markdown.Open always parses."
	if err != nil {
		return CreateNoteResult{}, err
	}
	// gomutants:disable-next-line BRANCH_IF reason="Freshly created document frontmatter always accepts SetScalar."
	if err := doc.SetScalar(typeKey, string(kind)); err != nil {
		return CreateNoteResult{}, err
	}
	// gomutants:disable-next-line BRANCH_IF reason="Freshly created document frontmatter always accepts SetList."
	if err := doc.SetList(fieldsKey, in.Fields); err != nil {
		return CreateNoteResult{}, err
	}
	// A note is shown by its `title`, else by its filename, so the key is
	// written only where the filename cannot carry the whole name.
	if !exact {
		// gomutants:disable-next-line BRANCH_IF reason="Freshly created document frontmatter always accepts SetTitle."
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

	// The file exists from here on, so the result reports its path whether or not
	// the index update succeeded.
	made := CreateNoteResult{Path: path, ID: identifier, Title: title}
	if u.index == nil {
		return made, nil
	}
	return made, edit.WrapIndexOutdated(u.index(ctx, v, []string{path}), path)
}

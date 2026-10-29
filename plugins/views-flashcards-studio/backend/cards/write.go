package cards

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Write puts a deck or a card model back where it came from.
//
// A deck and a card model are notes, so this is a note's write with a deck's bound
// on it: the body is replaced whole, and a file that has changed since the
// caller read it is left alone with ports.ErrStale. The frontmatter is the
// person's, apart from the key a card model declares its fields in and the
// identifier an edit stamps on a note carrying none.
type Write struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	// links resolves the card model a card's wikilink names, which determines
	// which of the card's fields is first.
	links ports.LinkQueries
	index note.Levels
	// now is when this is happening. An identifier written here carries it.
	now ports.Clock
}

// NewWrite is what a deck or a card model goes back through: the vault it is read
// and written through, where the wikilink each card names its card model by lands,
// what brings the file level in the index, and what time it is.
func NewWrite(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	links ports.LinkQueries,
	index note.Levels,
	now ports.Clock,
) Write {
	return Write{readers: readers, writers: writers, links: links, index: index, now: now}
}

// WithIndex replaces the index function.
func (u Write) WithIndex(index note.Levels) Write {
	u.index = index
	return u
}

// which is what the caller presents at its next write, and the card id every card
// that carried none was given.
type WriteResult struct {
	FileStat domain.FileStat
	Given    []format.CardID
}

// Deck puts body in the deck at path.
//
// The deck is made whole on the way past: every card carrying no card id is given
// one, and every stale heading is put back in step. Every caller therefore gets
// the same file, and none of them has to know the rules.
func (u Write) Deck(
	ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat,
) (WriteResult, error) {
	if err := note.CheckSize(path, len(body), MaxBytes); err != nil {
		return WriteResult{}, err
	}
	whole, provided, err := u.createWholeBody(ctx, v, path, body)
	if err != nil {
		return WriteResult{}, err
	}
	pos, err := u.createNoteWriter().Execute(ctx, v, path, whole, stat)
	if err != nil && !errors.Is(err, note.ErrUnlevelled) {
		return WriteResult{}, err
	}
	// levelling that failed, and the caller can tell the two apart.
	return WriteResult{FileStat: pos, Given: provided}, err
}

// createWholeBody returns the body with a card id on every card, and the card ids
// that were assigned. The card models are read against the body being written, so a
// card whose wikilink the caller changed is resolved against the card model the new wikilink names.
func (u Write) createWholeBody(
	ctx context.Context, v domain.Vault, path, body string,
) (string, []format.CardID, error) {
	read := NewRead(u.readers, u.links)
	delta, err := read.GetCardCardModels(ctx, v, path, format.ReadDeck(domain.Note{Body: body}))
	if err != nil {
		return "", nil, err
	}
	whole, provided := format.Whole(body, delta, cardid.New)
	return whole, provided, nil
}

// CardModel puts the templates and the fields into the card model at path. A card model is
// a note and is bounded as one.
//
// The templates are the body and the fields are one key of the frontmatter, and
// both are the one file, so one write carries both.
//
// The read, the change and the write are made under this vault's write lock,
// from before the read until after the file is replaced.
func (u Write) CardModel(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	stat domain.FileStat,
) (domain.FileStat, error) {
	if err := note.CheckSize(path, len(body), note.MaxBytes); err != nil {
		return domain.FileStat{}, err
	}
	if markdown.IsFrontmatterStart(body) {
		return domain.FileStat{}, note.ErrBodyUnwritable
	}

	e := note.NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat, e.Bound = stat, note.MaxBytes
	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		if err := doc.SetBody(body); err != nil {
			return err
		}
		// A card model already declaring these, in this order, keeps the bytes the
		// person wrote them as.
		if declared, _ := doc.List(fieldsKey); !slices.Equal(declared, fields) {
			return doc.SetList(fieldsKey, fields)
		}
		return nil
	})
}

// read at.
func (u Write) createNoteWriter() note.Write {
	writing := note.NewWrite(u.readers, u.writers, u.index, u.now)
	writing.Bound = MaxBytes
	return writing
}

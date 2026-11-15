package cards

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

// Write puts a deck or a card model back where it came from.
// It is a note's write with a deck's bound: the body is replaced whole, and a file changed since
// the caller read it is left alone with ports.ErrStale. The frontmatter is the person's, apart
// from the key a card model declares its fields in and the identifier an edit stamps on a note
// carrying none.
type Write struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	links   ports.LinkQueries
	index   note.IndexUpdater
	now     ports.Clock
}

// NewWrite is what a deck or a card model goes back through: the vault it is read
// and written through, where the wikilink each card names its card model by lands,
// what updates the index for the file, and what time it is.
func NewWrite(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	links ports.LinkQueries,
	index note.IndexUpdater,
	now ports.Clock,
) Write {
	return Write{readers: readers, writers: writers, links: links, index: index, now: now}
}

// WriteResult is the file stat of a written deck, which is what the caller presents at its next write,
// and the card id every card that carried none was given.
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
	if err != nil && !errors.Is(err, note.ErrIndexOutdated) {
		return WriteResult{}, err
	}
	// index update that failed, and the caller can tell the two apart.
	return WriteResult{FileStat: pos, Given: provided}, err
}

func (u Write) createWholeBody(
	ctx context.Context, v domain.Vault, path, body string,
) (string, []format.CardID, error) {
	read := NewRead(u.readers, u.links)
	delta, err := read.GetCardCardModels(ctx, v, path, format.ReadDeck(newBodyNote(body)))
	if err != nil {
		return "", nil, err
	}
	whole, provided := format.Whole(body, delta, createCardID)
	return whole, provided, nil
}

// CardModel puts the templates and the fields into the card model at path, bounded as a note.
// The templates are the body and the fields are one key of the frontmatter, so one write carries
// both. The read, the change and the write are made under this vault's write lock, from before
// the read until after the file is replaced.
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

	e, getReleaseError := note.NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat, e.Bound = stat, note.MaxBytes
	written, err := e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		// gomutants:disable-next-line BRANCH_IF,RETURN_ERROR_NIL reason="SetBody on an already opened document with non-frontmatter body always succeeds."
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
	return written, errors.Join(err, getReleaseError())
}

func (u Write) createNoteWriter() note.Write {
	writing := note.NewWrite(u.readers, u.writers, u.index, u.now)
	writing.Bound = MaxBytes
	return writing
}

func newBodyNote(body string) domain.Note {
	n := emptyNote()
	n.Body = body
	return n
}

package cards

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Write puts a deck or a stencil back where it came from.
//
// A deck and a stencil are notes, so this is a note's write with a deck's bound
// on it: the body is replaced whole, and a file that has changed since the
// caller read it is left alone with port.ErrChanged. The frontmatter is the
// person's, apart from the one key a stencil declares its fields in.
type Write struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	// Links answers where the wikilink a card names its stencil by lands, which
	// is what says which of the card's fields is first.
	Links port.LinkQueries
	Index note.Levels
	// Now is when this is happening. An identifier written here carries it.
	Now port.Clock
}

// NewWrite is what a deck or a stencil goes back through: the vault it is read
// and written through, where the wikilink each card names its stencil by lands,
// what brings the file level in the index, and what time it is.
//
// All five are named here because a write short of any one of them puts the
// file back and leaves something behind it — a card cut by the wrong stencil,
// a deck the vault cannot find, or a mark minted off the machine's clock.
func NewWrite(
	readers port.VaultReaders,
	writers port.VaultWriters,
	links port.LinkQueries,
	index note.Levels,
	now port.Clock,
) Write {
	return Write{Readers: readers, Writers: writers, Links: links, Index: index, Now: now}
}

// WriteResult is what a write of a deck left behind: what the file now stands at,
// which is what the caller presents at its next write, and the mark every card
// that carried none was given.
type WriteResult struct {
	Fingerprint domain.Fingerprint
	Minted      []format.MintedMark
}

// Deck puts body in the deck at path.
//
// The deck is made whole on the way past: every card carrying no mark is given
// one, and every stale heading is put back in step. Every caller therefore gets
// the same file, and none of them has to know the rules.
func (u Write) Deck(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.Fingerprint,
) (WriteResult, error) {
	if err := note.Bounded(path, len(body), MaxBytes); err != nil {
		return WriteResult{}, err
	}
	whole, minted, err := u.whole(ctx, v, path, body)
	if err != nil {
		return WriteResult{}, err
	}
	at, err := u.note().Execute(ctx, v, path, whole, fingerprint)
	if err != nil && !errors.Is(err, note.ErrUnlevelled) {
		return WriteResult{}, err
	}
	// The file is on disk, so what it now stands at comes back beside a
	// levelling that failed, and the caller can tell the two apart.
	return WriteResult{Fingerprint: at, Minted: minted}, err
}

// whole is the body every card of which has been made whole, and the marks that
// took. The stencils are read against the body being written, so a card whose
// wikilink the caller has just changed is cut by the stencil it now names.
func (u Write) whole(
	ctx context.Context, v domain.Vault, path, body string,
) (string, []format.MintedMark, error) {
	read := Read{Readers: u.Readers, Links: u.Links}
	by, err := read.Cutting(ctx, v, path, format.ReadDeck(domain.Note{Body: body}))
	if err != nil {
		return "", nil, err
	}
	whole, minted, err := format.Whole(body, by, cardid.New)
	if err != nil {
		return "", nil, fmt.Errorf("make %s whole: %w", path, err)
	}
	return whole, minted, nil
}

// Stencil puts the faces and the fields into the stencil at path. A stencil is
// a note and is bounded as one.
//
// The faces are the body and the fields are one key of the frontmatter, and
// both are the one file, so one write carries both.
func (u Write) Stencil(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if err := note.Bounded(path, len(body), note.MaxBytes); err != nil {
		return domain.Fingerprint{}, err
	}
	return u.stencil(ctx, v, path, body, fields, fingerprint)
}

// stencil is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (u Write) stencil(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if markdown.OpensFrontmatter(body) {
		return domain.Fingerprint{}, note.ErrBodyRefused
	}

	e := note.NewEditing(u.Readers, u.Writers, u.Index, u.Now)
	e.Fingerprint, e.Bound = fingerprint, note.MaxBytes
	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		doc.SetBody(body)
		// A stencil already declaring these, in this order, keeps the bytes the
		// person wrote them as.
		if declared, _ := doc.List(fieldsKey); !slices.Equal(declared, fields) {
			return doc.SetList(fieldsKey, fields)
		}
		return nil
	})
}

// note is the writer a deck goes to disk through, held to the size a deck is
// read at.
func (u Write) note() note.Write {
	writing := note.NewWrite(u.Readers, u.Writers, u.Index, u.Now)
	writing.Bound = MaxBytes
	return writing
}

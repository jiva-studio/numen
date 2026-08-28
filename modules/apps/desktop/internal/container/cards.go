package container

import (
	"context"
	"strings"
	"time"

	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/cards"
)

// Cards is everything that acts on the two notes a flashcard is made of. Both
// the window and the tools an agent calls are served the same set, so what one
// of them refuses the other refuses.
type Cards struct {
	Read   cards.Read
	List   cards.List
	Write  cards.Write
	Create cards.Create
	Rename cards.RenameField
}

// Cards builds them against this installation's vault readers and writers.
//
// Index brings what a write touched up to date before it answers, so a caller
// that makes a deck and lists the vault's stencils in the next breath finds
// what it made.
func (c Config) Cards(
	notes port.NoteQueries,
	links port.LinkQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
) Cards {
	readers := c.VaultReaders()
	writers := c.VaultWriters()
	return Cards{
		Read: cards.Read{Readers: readers, Links: links},
		List: cards.List{Readers: readers, Notes: notes},
		Write: cards.Write{
			Readers: readers, Writers: writers, Index: index, Now: time.Now,
		},
		Create: cards.Create{
			Writers: writers, Index: index, Extension: c.NoteExtension(), Now: time.Now,
		},
		Rename: cards.RenameField{
			Readers: readers, Writers: writers, Notes: notes, Links: links, Index: index,
		},
	}
}

// DeckBody is the markdown a deck of these cards is written as: the preamble as
// it arrived, each card laid down by the format itself, and the tail verbatim
// below the last value.
//
// The cards are written into a file of no cards, so a card nobody touched comes
// out as the bytes it went in as.
func DeckBody(preamble string, cs []format.Card, tail string) (string, error) {
	scratch, err := format.OpenDeck(markdown.Create("", preamble))
	if err != nil {
		return "", err
	}
	for _, card := range cs {
		if err := scratch.AddCard(card); err != nil {
			return "", err
		}
	}
	body, err := below(scratch.Bytes())
	if err != nil || len(cs) == 0 {
		return body, err
	}
	// The tail opens with the break that ends the last value, so the break the
	// last card was written with goes.
	return strings.TrimRight(body, "\n") + tail, nil
}

// StencilBody is the markdown these faces are written as, in the order they are
// to stand in the note.
//
// The faces are written into a stencil of no faces, one after another, so every
// face a caller gave stands in the file and two of one name are two faces.
func StencilBody(fs []format.Face) (string, error) {
	scratch, err := format.OpenStencil(markdown.Create("", ""))
	if err != nil {
		return "", err
	}
	for _, face := range fs {
		if err := scratch.AddFace(face); err != nil {
			return "", err
		}
	}
	return below(scratch.Bytes())
}

// below is what stands under the frontmatter of a file just written.
func below(raw []byte) (string, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return "", err
	}
	return doc.Body(), nil
}

// NoteExtension is what a note this vault holds is filed under. Empty is
// markdown.
func (c Config) NoteExtension() string {
	if len(c.Extensions) > 0 {
		return c.Extensions[0]
	}
	return ""
}

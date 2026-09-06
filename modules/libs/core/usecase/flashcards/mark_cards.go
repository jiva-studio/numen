package flashcards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// MarkCards gives a mark to every card of a vault that carries none.
//
// A card typed by hand carries no mark until the application writes its file,
// and a card with no mark has nothing an answer can be recorded against. A deck
// holding one is written, which mints a mark for every card in it.
//
// This is the one thing flashcards writes into a vault, and it is done when a
// person sits down to that vault — not to every vault the installation holds,
// and not for the counting of what is owed.
type MarkCards struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Notes   port.NoteQueries
	Links   port.LinkQueries
	// Index brings what a write touched up to date. A build holding none leaves
	// the index to the next scan.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	// Now is when this is happening. A mark written here carries it.
	Now port.Clock
}

// NewMarkCards is what a vault's cards are given marks through: the vault the
// decks are read out of and written back to, what says which of its notes are
// decks, where the wikilink a card names its stencil by lands, what brings a
// written deck level in the index, and what time it is.
//
// All six are named here because a marking short of any one of them leaves a
// card with no mark, which is a card the session after it cannot ask, or a mark
// minted off the machine's clock rather than this installation's.
func NewMarkCards(
	readers port.VaultReaders,
	writers port.VaultWriters,
	notes port.NoteQueries,
	links port.LinkQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
	now port.Clock,
) MarkCards {
	return MarkCards{
		Readers: readers, Writers: writers, Notes: notes, Links: links,
		Index: index, Now: now,
	}
}

// MarkCardsResult is what the marking came to: the decks it could not write.
type MarkCardsResult struct {
	// Unwritten are the paths of the decks holding a card with no mark that
	// could not be given one. Their cards are left out of this session.
	Unwritten []string
}

// Execute writes every deck of the vault that holds a card with no mark.
//
// A deck that could not be written is left as it is and is not an error: the
// editor may be saving it, and the vault's write lock lives in one process. Its
// cards are left out of this session and marked at the next, and it is named in
// what comes back so that a person is told which deck that was.
func (u MarkCards) Execute(ctx context.Context, v domain.Vault) (MarkCardsResult, error) {
	paths, err := u.Notes.OfType(ctx, v.ID, domain.TypeDeck)
	if err != nil {
		return MarkCardsResult{}, err
	}

	read := cards.NewRead(u.Readers, u.Links)
	write := cards.NewWrite(u.Readers, u.Writers, u.Links, u.Index, u.Now)
	var out MarkCardsResult
	for _, path := range paths {
		deck, err := read.Deck(ctx, v, path)
		if err != nil || deck.Outcome != note.Ok || !unmarked(deck.Body) {
			continue
		}

		// The body goes back exactly as it was read. What the write is for is
		// the deck being made whole on the way past, which is where a mark is
		// minted.
		body, err := format.DeckBody(deck.Body)
		if err != nil {
			out.Unwritten = append(out.Unwritten, path)
			continue
		}
		// A deck the index could not be brought level with carries its marks all
		// the same, and its cards stand in this session.
		if _, err := write.Deck(ctx, v, path, body, deck.Fingerprint); err != nil &&
			!errors.Is(err, note.ErrUnlevelled) {
			out.Unwritten = append(out.Unwritten, path)
		}
	}
	return out, nil
}

// unmarked reports whether any card of a deck carries no mark.
func unmarked(d format.Deck) bool {
	for _, c := range d.Cards {
		if c.Mark == "" {
			return true
		}
	}
	return false
}

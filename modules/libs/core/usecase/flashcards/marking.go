package flashcards

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// Marking gives a mark to every card of a vault that carries none.
//
// A card typed by hand carries no mark until the application writes its file,
// and a card with no mark has nothing an answer can be recorded against. A deck
// holding one is written, which mints a mark for every card in it.
//
// This is the one thing flashcards writes into a vault, and it is done when a
// person sits down to that vault — not to every vault the installation holds,
// and not for the counting of what is owed.
type Marking struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Notes   port.NoteQueries
	Links   port.LinkQueries
	// Index brings what a write touched up to date. A build holding none leaves
	// the index to the next scan.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	Now   func() time.Time
}

// MarkingResult is what the marking came to: the decks it could not write.
type MarkingResult struct {
	// Unwritten are the paths of the decks holding a card with no mark that
	// could not be given one. Their cards are left out of this sitting.
	Unwritten []string
}

// Execute writes every deck of the vault that holds a card with no mark.
//
// A deck that could not be written is left as it is and is not an error: the
// editor may be saving it, and the vault's write lock lives in one process. Its
// cards are left out of this sitting and marked at the next, and it is named in
// what comes back so that a person is told which deck that was.
func (u Marking) Execute(ctx context.Context, v domain.Vault) (MarkingResult, error) {
	paths, err := u.Notes.OfType(ctx, v.ID, domain.TypeDeck)
	if err != nil {
		return MarkingResult{}, err
	}

	read := cards.Read{Readers: u.Readers, Links: u.Links}
	write := cards.Write{
		Readers: u.Readers, Writers: u.Writers, Links: u.Links, Index: u.Index, Now: u.Now,
	}
	var out MarkingResult
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
		// the same, and its cards stand in this sitting.
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

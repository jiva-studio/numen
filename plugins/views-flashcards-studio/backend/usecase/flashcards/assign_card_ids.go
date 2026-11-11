package flashcards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

// AssignCardIDs gives a card id to every card of a vault that carries none.
// A card typed manually has no card id until the application writes its file, and a card without
// one cannot have an answer recorded against it. This is the only write flashcards makes into a
// vault, and it is made when a session starts, not when due cards are counted.
type AssignCardIDs struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	notes   ports.TypePathQueries
	links   ports.LinkQueries
	index   func(ctx context.Context, v domain.Vault, paths []string) error
	now     ports.Clock
}

// NewAssignCardIDs constructs an AssignCardIDs from the vault readers and writers
// for reading and rewriting decks, the note queries that identify which notes are
// decks, the link queries that resolve the wikilink naming a card's card model,
// the index function that updates the index after a deck is written, and the
// clock.
func NewAssignCardIDs(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	notes ports.TypePathQueries,
	links ports.LinkQueries,
	index func(ctx context.Context, v domain.Vault, paths []string) error,
	now ports.Clock,
) AssignCardIDs {
	return AssignCardIDs{
		readers: readers, writers: writers, notes: notes, links: links,
		index: index, now: now,
	}
}

// WithIndex returns a copy of AssignCardIDs with the given index function.
func (u AssignCardIDs) WithIndex(index func(ctx context.Context, v domain.Vault, paths []string) error) AssignCardIDs {
	u.index = index
	return u
}

// AssignCardIDsResult is what the assigning of card ids came to: the decks it could not write.
type AssignCardIDsResult struct {
	Unwritten []string
}

// Execute writes every deck of the vault that holds a card with no card id.
//
// A deck that could not be written is left as it is and is not an error: the
// editor may be saving it, and the vault's write lock lives in one process. Its
// cards are left out of this session and marked at the next, and it is named in
func (u AssignCardIDs) Execute(ctx context.Context, v domain.Vault) (AssignCardIDsResult, error) {
	if u.notes == nil {
		return AssignCardIDsResult{Unwritten: nil}, nil
	}
	paths, err := u.notes.GetPathsOfType(ctx, v.ID, format.TypeDeck)
	if err != nil {
		return AssignCardIDsResult{}, err
	}

	read := cards.NewRead(u.readers, u.links)
	write := cards.NewWrite(u.readers, u.writers, u.links, u.index, u.now)
	var out AssignCardIDsResult
	for _, path := range paths {
		deck, err := read.Deck(ctx, v, path)
		if err != nil || deck.Outcome != note.Ok || !hasCardWithoutID(deck.Body) {
			continue
		}

		// The body goes back exactly as it was read. What the write is for is
		// the deck being made whole on the way past, which is where a card id is
		// minted.
		body, err := format.DeckBody(deck.Body)
		// gomutants:disable-next-line BRANCH_IF reason="Parsed decks always have valid section indices and card model links."
		if err != nil {
			out.Unwritten = append(out.Unwritten, path)
			continue
		}
		// A deck the index could not be updated for carries its card ids all
		// the same, and its cards stand in this session.
		if _, err := write.Deck(ctx, v, path, body, deck.FileStat); err != nil &&
			!errors.Is(err, note.ErrIndexOutdated) {
			out.Unwritten = append(out.Unwritten, path)
		}
	}
	return out, nil
}

func hasCardWithoutID(d format.Deck) bool {
	for _, c := range d.Cards {
		if c.CardID == "" {
			return true
		}
	}
	return false
}

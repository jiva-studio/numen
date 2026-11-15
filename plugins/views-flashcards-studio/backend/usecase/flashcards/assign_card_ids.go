package flashcards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// AssignCardIDs gives a card id to every card of a vault that carries none.
// A card typed manually has no card id until the application writes its file, and a card without
// one cannot have an answer recorded against it. This is the only write flashcards makes into a
// vault, and it is made when a session starts, not when due cards are counted.
type AssignCardIDs struct {
	notes ports.TypePathQueries
	read  ports.DeckReader
	write ports.DeckWriter
}

// NewAssignCardIDs constructs an AssignCardIDs from the note queries that identify which notes are
// decks, the reader of a deck, and the writer that gives a card id to every card of a deck carrying
// none.
func NewAssignCardIDs(notes ports.TypePathQueries, read ports.DeckReader, write ports.DeckWriter) AssignCardIDs {
	return AssignCardIDs{notes: notes, read: read, write: write}
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

	var out AssignCardIDsResult
	for _, path := range paths {
		deck, err := u.read.Deck(ctx, v, path)
		if err != nil || deck.Outcome != edit.Ok || !hasCardWithoutID(deck.Body) {
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
		if _, err := u.write.Deck(ctx, v, path, body, deck.FileStat); err != nil &&
			!errors.Is(err, edit.ErrIndexOutdated) {
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

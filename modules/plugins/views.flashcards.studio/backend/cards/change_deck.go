package cards

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
)

// deckChange is what a change wrote: the deck as it became and the marks given to cards that carried none.
type deckChange struct {
	result EditResult
	given  []format.CardMark
}

// changeDeck splices one change into a deck. Every byte the change does not name is written as it arrived.
func changeDeck(
	ctx context.Context, read Read, write Write, v domain.Vault, pos DeckVersion,
	change func(contents DeckContents, file *format.DeckFile) error,
) (deckChange, error) {
	contents, err := readExistingDeck(ctx, read, v, pos.Path)
	if err != nil {
		return deckChange{}, err
	}
	file := format.OpenDeckBody(contents.Raw)
	if err := change(contents, file); err != nil {
		return deckChange{}, err
	}
	wrote, err := write.Deck(ctx, v, pos.Path, file.Body(), pos.Fingerprint)
	if err != nil && !errors.Is(err, note.ErrUnlevelled) {
		return deckChange{}, err
	}
	return deckChange{
		result: EditResult{Fingerprint: wrote.Fingerprint, Cards: len(file.Deck(domain.Fingerprint{}).Cards)},
		given:  wrote.Given,
	}, nil
}
